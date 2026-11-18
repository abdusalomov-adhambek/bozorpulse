package forecast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
)

// OpenMeteo is a keyless weather provider. Results are cached for 30 minutes
// per region so a shortlist of offers does not trigger repeated calls.
type OpenMeteo struct {
	client *http.Client
	ttl    time.Duration

	mu    sync.Mutex
	cache map[string]cachedWeather
}

type cachedWeather struct {
	info domain.WeatherInfo
	at   time.Time
}

var regionCoords = map[string][2]float64{
	"reg-01": {41.2995, 69.2401}, // Tashkent
	"reg-02": {39.6270, 66.9750}, // Samarkand
	"reg-03": {39.7682, 64.6221}, // Bukhara
	"reg-04": {40.8139, 72.3200}, // Andijan
	"reg-05": {40.3889, 71.7889}, // Fergana
}

func NewOpenMeteo(client *http.Client) *OpenMeteo {
	if client == nil {
		client = &http.Client{Timeout: 4 * time.Second}
	}
	return &OpenMeteo{client: client, ttl: 30 * time.Minute, cache: map[string]cachedWeather{}}
}

func (p *OpenMeteo) Current(ctx context.Context, regionID string) (domain.WeatherInfo, error) {
	info := domain.WeatherInfo{Region: regionName(regionID), Source: "open-meteo"}
	coords, ok := regionCoords[regionID]
	if !ok {
		return info, fmt.Errorf("no coordinates for region %q", regionID)
	}

	p.mu.Lock()
	if hit, ok := p.cache[regionID]; ok && time.Since(hit.at) < p.ttl {
		p.mu.Unlock()
		return hit.info, nil
	}
	p.mu.Unlock()

	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%.4f", coords[0]))
	q.Set("longitude", fmt.Sprintf("%.4f", coords[1]))
	q.Set("current", "temperature_2m,precipitation")
	q.Set("timezone", "Asia/Tashkent")
	q.Set("forecast_days", "1")
	endpoint := "https://api.open-meteo.com/v1/forecast?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return info, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return info, fmt.Errorf("open-meteo: status %d", resp.StatusCode)
	}
	var wire struct {
		Current struct {
			Temperature2m float64 `json:"temperature_2m"`
			Precipitation float64 `json:"precipitation"`
			Time          string  `json:"time"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return info, err
	}
	info.TemperatureC = wire.Current.Temperature2m
	info.PrecipitationMM = wire.Current.Precipitation
	info.ObservedAt = wire.Current.Time

	p.mu.Lock()
	p.cache[regionID] = cachedWeather{info: info, at: time.Now()}
	p.mu.Unlock()
	return info, nil
}

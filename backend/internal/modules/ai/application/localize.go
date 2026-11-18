package application

import (
	"strings"
	"unicode"
)

// detectLanguage determines the language of the question itself. The UI's
// selected locale is only a fallback when the text gives no clear signal, so
// a Russian question typed while "English" is selected still gets Russian.
func detectLanguage(message, fallback string) string {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return normalizeLocale(fallback)
	}

	hasCyrillic := false
	for _, r := range lower {
		if unicode.Is(unicode.Cyrillic, r) {
			hasCyrillic = true
			break
		}
	}
	if hasCyrillic {
		// These letters are used by Uzbek Cyrillic but not by Russian.
		for _, ch := range []rune{'ў', 'қ', 'ҳ', 'ғ'} {
			if strings.ContainsRune(lower, ch) {
				return "uz"
			}
		}
		return "ru"
	}

	uzWords := []string{"soat", "sotib", "olmoqchi", "olmoqchiman", "olish", "kerak", "qancha", "narxi",
		"qayerdan", "nemadan", "mahsulot", "yetkazib", "assalom", "rahmat", "menga", "men", "biz",
		"uchun", "qanday", "nima", "bormi", "qilmoqchi", "kun", "yil", "shahar", "ombor", "arzon", "eng"}
	enWords := []string{"the", "is", "are", "how", "much", "what", "where", "who", "need", "want", "buy",
		"price", "cost", "supplier", "product", "order", "status", "please", "i", "you", "can", "do",
		"my", "of", "for", "in", "to", "and", "cheaper", "faster", "recommend", "compare", "show", "find"}
	words := strings.FieldsFunc(lower, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	uzScore, enScore := 0, 0
	for _, w := range words {
		for _, u := range uzWords {
			if w == u {
				uzScore++
			}
		}
		for _, e := range enWords {
			if w == e {
				enScore++
			}
		}
	}
	if strings.Contains(lower, "o'") || strings.Contains(lower, "g'") {
		uzScore += 2
	}
	if uzScore > enScore {
		return "uz"
	}
	if enScore > uzScore {
		return "en"
	}
	return normalizeLocale(fallback)
}

func normalizeLocale(locale string) string {
	switch {
	case strings.HasPrefix(strings.ToLower(locale), "uz"), strings.HasPrefix(strings.ToLower(locale), "уzb"):
		return "uz"
	case strings.HasPrefix(strings.ToLower(locale), "ru"):
		return "ru"
	default:
		return "en"
	}
}

func languageName(locale string) string {
	switch normalizeLocale(locale) {
	case "uz":
		return "UZBEK"
	case "ru":
		return "RUSSIAN"
	default:
		return "ENGLISH"
	}
}

// replyStrings holds every deterministic template. The LLM path is guided by
// the system prompt; these strings cover the rule-based and shopping paths.
type replyStrings struct {
	productClarify  string
	quantityClarify string
	noOffers        string
	topOptions      string // %d = number of options
	topOption       string
	fastestOptions  string // %d = number of options
	fastestOption   string
	header          string // %s label, %d qty, %s unit
	line            string // %d rank, %s name, %s supplier, %.2f price, %s unit, %.2f total, %d stock, %d min days, %d max days
	forecast        string // %d horizon, %.2f predicted, %s unit, %.2f low, %.2f high, %s basis
	forecastGeneric string // %d horizon, %.2f predicted, %.2f low, %.2f high, %s basis
	unitNote        string // %s product, %s product unit, %s label, %s requested unit
	cheaper         string
	faster          string
	needCement      string
	unitsWord       string
	viewOffer       string
	addToBasket     string
	goToOrders      string
	outOfScope      string
	onboarding      string
	smallTalk       string
	orderAsk        string
	orderSuggest    string
	orderNotFound   string
	orderStatus     string // %s id, %s status, %.2f total
	supplierAsk     string
	offersNotFound  string
	searchError     string
	priceError      string
	supplierFound   string // %d count
	productFound    string // %d count
	productLine     string // %d rank, %s name, %.2f price, %s unit
	marketNoData    string
	marketStats     string // %d count, %.2f min, %.2f max, %.2f median, %.2f avg
	buyingFallback  string
	genericFallback string
	suggestions     []string
	marketSuggest   []string
	supplierSuggest []string
	defaultSuggest  []string
}

var replyTexts = map[string]replyStrings{
	"en": {
		productClarify:  "What product are you looking for? Tell me the product and roughly how much you need.",
		quantityClarify: "Roughly how much do you need? Tell me the quantity and I'll compare real offers for you.",
		noOffers:        "I couldn't find offers for %q. Check the product name or try a different amount.",
		topOptions:      "Top %d options", topOption: "Top option",
		fastestOptions: "Fastest of %d options", fastestOption: "Fastest option",
		header:          "%s for about %d %s:",
		line:            "%d. %s — %s, %.2f per %s (%.2f total), %d in stock, %d–%d days delivery.",
		forecast:        "Price outlook (next %d days): about %.2f per %s (range %.2f–%.2f). %s",
		forecastGeneric: "Price outlook (next %d days): about %.2f (range %.2f–%.2f). %s",
		unitNote:        "%s is sold per %s, so the amounts below are in %s (the catalog cannot convert %s). ",
		cheaper:         "And the cheaper one?", faster: "Which is faster?",
		needCement: "I need 100 bags of cement", unitsWord: "units",
		viewOffer: "View offer", addToBasket: "Add to basket", goToOrders: "Go to orders",
		outOfScope:      "Payments, delivery tracking and automatic purchasing are not part of this platform yet. I can help you compare supplier offers and create an order instead.",
		onboarding:      "Register with a username, password, INN and bank account. Your account starts as Waiting; an admin approves it before your cabinet becomes available.",
		smallTalk:       "Hello! I can help with supplier prices, finding suppliers, order status and creating orders.",
		orderAsk:        "Which order would you like to check? Please tell me the order number.",
		orderSuggest:    "Status of order 1",
		orderNotFound:   "I couldn't find that order among your orders.",
		orderStatus:     "Order #%s is %s with a total of %.2f.",
		supplierAsk:     "Which product are you interested in? Tell me the product name and I'll find suppliers for it.",
		offersNotFound:  "I couldn't find comparable offers right now.",
		searchError:     "I couldn't search the catalog right now.",
		priceError:      "I couldn't find comparable market prices right now.",
		supplierFound:   "Found %d comparable supplier offers:",
		productFound:    "Found %d products matching your request:",
		productLine:     "%d. %s — %.2f per %s",
		marketNoData:    "There are not enough comparable price samples yet to quote a market level. Please compare available supplier offers directly.",
		marketStats:     "Across %d comparable samples, observed prices range from %.2f to %.2f with a median of %.2f and an average of %.2f. These are seller-provided prices, not independently verified market observations.",
		buyingFallback:  "I can help with market prices, finding suppliers, product search, your orders and what to buy. What would you like to know?",
		genericFallback: "I can help with market prices, finding suppliers, product search, your orders and what to buy. What would you like to know?",
		suggestions:     []string{"And the cheaper one?", "Which is faster?"},
		marketSuggest:   []string{"Who sells cement?", "What should I buy?"},
		supplierSuggest: []string{"How much is cement?", "How do I create an order?"},
		defaultSuggest:  []string{"How much is cement?", "Who sells cement?", "What should I buy?"},
	},
	"uz": {
		productClarify:  "Qaysi mahsulot kerak? Mahsulot nomini va taxminan miqdorini ayting.",
		quantityClarify: "Taxminan qancha kerak? Miqdorni ayting — haqiqiy takliflarni solishtirib beraman.",
		noOffers:        "%q bo'yicha taklif topolmadim. Mahsulot nomini tekshiring yoki boshqa miqdor sinang.",
		topOptions:      "Eng yaxshi %d variant", topOption: "Eng yaxshi variant",
		fastestOptions: "Eng tez %d variant", fastestOption: "Eng tez variant",
		header:          "%s — taxminan %d %s uchun:",
		line:            "%d. %s — %s, %.2f %s uchun (jami %.2f), zaxirada %d, yetkazib berish %d–%d kun.",
		forecast:        "Narx prognozi (keyingi %d kun): %.2f %s uchun (oraliq %.2f–%.2f). %s",
		forecastGeneric: "Narx prognozi (keyingi %d kun): taxminan %.2f (oraliq %.2f–%.2f). %s",
		unitNote:        "%s %s bilan sotiladi, shuning uchun quyidagi miqdorlar %s da (katalog %s ni o'zgartira olmaydi). ",
		cheaper:         "Arzonrog'i bormi?", faster: "Qaysisi tezroq?",
		needCement: "Menga 100 qop sement kerak", unitsWord: "dona",
		viewOffer: "Taklifni ko'rish", addToBasket: "Savatga qo'shish", goToOrders: "Buyurtmalarga o'tish",
		outOfScope:      "To'lovlar, yetkazib berishni kuzatish va avtomatik xarid qilish hozircha mavjud emas. Men yetkazib beruvchi takliflarini solishtirib, buyurtma yaratishga yordam bera olaman.",
		onboarding:      "Foydalanuvchi nomi, parol, INN va bank hisobi bilan ro'yxatdan o'ting. Hisob avval Kutish holatida bo'ladi; administrator tasdiqlagach kabinet ishga tushadi.",
		smallTalk:       "Assalomu alaykum! Narxlar, yetkazib beruvchilar, buyurtma holati va buyurtma yaratishda yordam bera olaman.",
		orderAsk:        "Qaysi buyurtmani tekshirmoqchisiz? Buyurtma raqamini ayting.",
		orderSuggest:    "1-buyurtma holati",
		orderNotFound:   "Buyurtmalaringiz orasida buni topolmadim.",
		orderStatus:     "#%s buyurtma holati %s, jami %.2f.",
		supplierAsk:     "Qaysi mahsulot qiziqtiryapti? Mahsulot nomini ayting — yetkazib beruvchilarni topaman.",
		offersNotFound:  "Hozircha taqqoslanadigan takliflarni topolmadim.",
		searchError:     "Katalogda qidirib topolmadim.",
		priceError:      "Taqqoslanadigan bozor narxlarini topolmadim.",
		supplierFound:   "%d ta taqqoslanadigan yetkazib beruvchi taklifi topildi:",
		productFound:    "So'rovga mos %d ta mahsulot topildi:",
		productLine:     "%d. %s — %.2f %s uchun",
		marketNoData:    "Bozor narxi uchun yetarli taqqoslanadigan namunalar yo'q. Mavjud takliflarni o'zingiz solishtiring.",
		marketStats:     "%d ta taqqoslanadigan namuna bo'yicha narxlar %.2f dan %.2f gacha, mediana %.2f, o'rtacha %.2f. Bu sotuvchilar bergan narxlar, mustaqil bozor kuzatuvi emas.",
		buyingFallback:  "Bozor narxlari, yetkazib beruvchilar, mahsulot qidiruvi, buyurtmalar va nima sotib olish haqida yordam bera olaman. Nima bilmoqchisiz?",
		genericFallback: "Bozor narxlari, yetkazib beruvchilar, mahsulot qidiruvi, buyurtmalar va nima sotib olish haqida yordam bera olaman. Nima bilmoqchisiz?",
		suggestions:     []string{"Arzonrog'i bormi?", "Qaysisi tezroq?"},
		marketSuggest:   []string{"Sementni kim sotadi?", "Nima sotib olish kerak?"},
		supplierSuggest: []string{"Sement narxi qancha?", "Buyurtmani qanday yarataman?"},
		defaultSuggest:  []string{"Sement narxi qancha?", "Sementni kim sotadi?", "Nima sotib olish kerak?"},
	},
	"ru": {
		productClarify:  "Какой товар вам нужен? Назовите товар и примерное количество.",
		quantityClarify: "Примерно сколько нужно? Укажите количество — я сравню реальные предложения.",
		noOffers:        "Не нашёл предложений для %q. Проверьте название товара или попробуйте другое количество.",
		topOptions:      "Топ %d вариантов", topOption: "Лучший вариант",
		fastestOptions: "Самые быстрые варианты (%d)", fastestOption: "Самый быстрый вариант",
		header:          "%s на примерно %d %s:",
		line:            "%d. %s — %s, %.2f за %s (итого %.2f), в наличии %d, доставка %d–%d дн.",
		forecast:        "Прогноз цены (на %d дней): примерно %.2f за %s (диапазон %.2f–%.2f). %s",
		forecastGeneric: "Прогноз цены (на %d дней): примерно %.2f (диапазон %.2f–%.2f). %s",
		unitNote:        "%s продаётся по %s, поэтому суммы ниже указаны в %s (каталог не конвертирует %s). ",
		cheaper:         "Есть дешевле?", faster: "Какой быстрее?",
		needCement: "Мне 100 мешков цемента", unitsWord: "шт",
		viewOffer: "Смотреть предложение", addToBasket: "В корзину", goToOrders: "К заказам",
		outOfScope:      "Оплата, отслеживание доставки и автоматическая закупка пока не реализованы. Я могу сравнить предложения поставщиков и помочь оформить заказ.",
		onboarding:      "Зарегистрируйтесь с именем пользователя, паролем, ИНН и банковским счётом. Счёт начинается со статуса Ожидание; администратор подтвердит его перед открытием кабинета.",
		smallTalk:       "Здравствуйте! Помогу с ценами поставщиков, поиском поставщиков, статусом заказа и созданием заказа.",
		orderAsk:        "Какой заказ проверить? Укажите номер заказа.",
		orderSuggest:    "Статус заказа 1",
		orderNotFound:   "Не нашёл этот заказ среди ваших заказов.",
		orderStatus:     "Заказ №%s — %s, итого %.2f.",
		supplierAsk:     "Какой товар вас интересует? Назовите товар — найду поставщиков.",
		offersNotFound:  "Сейчас не нашёл сопоставимых предложений.",
		searchError:     "Не удалось выполнить поиск по каталогу.",
		priceError:      "Не удалось найти сопоставимые рыночные цены.",
		supplierFound:   "Найдено предложений поставщиков: %d",
		productFound:    "Найдено товаров по запросу: %d",
		productLine:     "%d. %s — %.2f за %s",
		marketNoData:    "Пока недостаточно сопоставимых данных о ценах. Сравните доступные предложения поставщиков напрямую.",
		marketStats:     "По %d сопоставимым наблюдениям цены составляют от %.2f до %.2f, медиана %.2f, среднее %.2f. Это цены, указанные продавцами, а не независимые рыночные наблюдения.",
		buyingFallback:  "Помогу с рыночными ценами, поиском поставщиков, поиском товаров, заказами и выбором покупки. Что вас интересует?",
		genericFallback: "Помогу с рыночными ценами, поиском поставщиков, поиском товаров, заказами и выбором покупки. Что вас интересует?",
		suggestions:     []string{"Есть дешевле?", "Какой быстрее?"},
		marketSuggest:   []string{"Кто продаёт цемент?", "Что купить?"},
		supplierSuggest: []string{"Сколько стоит цемент?", "Как создать заказ?"},
		defaultSuggest:  []string{"Сколько стоит цемент?", "Кто продаёт цемент?", "Что купить?"},
	},
}

// localizeReason translates the fixed reason codes produced by the ranking
// engines. Unknown reasons (e.g. a dynamic "Listing updated N days ago") keep
// their structure and only the known words are swapped.
func localizeReason(reason, locale string) string {
	reasons := map[string]map[string]string{
		"Lowest total for this quantity": {"uz": "Eng past jami narx", "ru": "Самая низкая итоговая цена"},
		"Fast delivery":                  {"uz": "Tez yetkazib berish", "ru": "Быстрая доставка"},
		"Enough stock available":         {"uz": "Zaxira yetarli", "ru": "Достаточно на складе"},
		"Comparable offer in the catalog": {"uz": "Katalogdagi mos taklif", "ru": "Сопоставимое предложение в каталоге"},
		"Price at or below the market median": {"uz": "Narx bozor medianidan past yoki teng", "ru": "Цена ниже или равна медиане рынка"},
		"Price close to the market median":    {"uz": "Narx bozor medianiga yaqin", "ru": "Цена близка к медиане рынка"},
		"Same region as your warehouse":       {"uz": "Omborga bir hududda", "ru": "Тот же регион, что и ваш склад"},
		"Verified supplier":                   {"uz": "Tasdiqlangan yetkazib beruvchi", "ru": "Проверенный поставщик"},
		"Limited comparable data — compare offers manually": {"uz": "Taqqoslanadigan ma'lumot oz — takliflarni o'zingiz solishtiring", "ru": "Мало сопоставимых данных — сравните предложения вручную"},
	}
	if tr, ok := reasons[reason]; ok {
		if localized, ok := tr[normalizeLocale(locale)]; ok {
			return localized
		}
	}
	loc := normalizeLocale(locale)
	if strings.HasPrefix(reason, "Listing updated") {
		switch loc {
		case "uz":
			return strings.Replace(reason, "Listing updated", "Ro'yxat yangilandi", 1)
		case "ru":
			return strings.Replace(reason, "Listing updated", "Объявление обновлено", 1)
		}
	}
	return reason
}

func localizeReasons(reasons []string, locale string) []string {
	out := make([]string, 0, len(reasons))
	for _, r := range reasons {
		out = append(out, localizeReason(r, locale))
	}
	return out
}

func text(locale string) replyStrings {
	if t, ok := replyTexts[normalizeLocale(locale)]; ok {
		return t
	}
	return replyTexts["en"]
}

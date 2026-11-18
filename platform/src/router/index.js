import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

import Login from '../views/auth/Login.vue'
import Register from '../views/auth/Register.vue'
import Waiting from '../views/auth/Waiting.vue'

import CabinetLayout from '../layouts/CabinetLayout.vue'
import ChatHistory from '../views/ChatHistory.vue'

import AdminDashboard from '../views/admin/Dashboard.vue'
import AdminSuppliers from '../views/admin/Suppliers.vue'
import AdminCustomers from '../views/admin/Customers.vue'
import AdminOrders from '../views/admin/Orders.vue'
import AdminCategories from '../views/admin/Categories.vue'
import AdminProducts from '../views/admin/Products.vue'
import AdminRegions from '../views/admin/Regions.vue'
import AdminPreviewSupplier from '../views/admin/PreviewSupplier.vue'
import AdminPreviewCustomer from '../views/admin/PreviewCustomer.vue'

import CustomerDashboard from '../views/customer/Dashboard.vue'
import CustomerOrders from '../views/customer/Orders.vue'
import CustomerSuppliers from '../views/customer/Suppliers.vue'
import CustomerPayments from '../views/customer/Payments.vue'
import CustomerWarehouses from '../views/customer/Warehouses.vue'
import CustomerMarket from '../views/customer/Market.vue'
import CustomerPlanning from '../views/customer/Planning.vue'
import CustomerProductDetail from '../views/customer/ProductDetail.vue'
import CustomerBasket from '../views/customer/Basket.vue'
import CustomerCabinet from '../views/customer/Cabinet.vue'

import SupplierDashboard from '../views/supplier/Dashboard.vue'
import SupplierOrders from '../views/supplier/Orders.vue'
import SupplierPayments from '../views/supplier/Payments.vue'
import SupplierProducts from '../views/supplier/Products.vue'
import SupplierDelivery from '../views/supplier/Delivery.vue'
import SupplierWarehouses from '../views/supplier/Warehouses.vue'
import SupplierCabinet from '../views/supplier/Cabinet.vue'

// Sidebar configuration, keyed by role. Consumed by CabinetLayout.
export const NAV_CONFIG = {
  admin: [
    { label: 'Dashboard', to: '/admin/dashboard', icon: 'dashboard' },
    { label: 'Suppliers', to: '/admin/suppliers', icon: 'suppliers' },
    { label: 'Customers', to: '/admin/customers', icon: 'customers' },
    { label: 'Orders', to: '/admin/orders', icon: 'orders' },
    { label: 'Chat history', to: '/admin/chat-history', icon: 'sparkle' },
    {
      label: 'Preview',
      icon: 'sparkle',
      children: [
        { label: 'Supplier cabinet', to: '/admin/preview/supplier' },
        { label: 'Customer cabinet', to: '/admin/preview/customer' }
      ]
    },
    {
      label: 'Settings',
      icon: 'settings',
      children: [
        { label: 'Product categories', to: '/admin/settings/categories' },
        { label: 'Products', to: '/admin/settings/products' },
        { label: 'Regions', to: '/admin/settings/regions' }
      ]
    }
  ],
  customer: [
    { label: 'Dashboard', to: '/customer/dashboard', icon: 'dashboard' },
    { label: 'Orders', to: '/customer/orders', icon: 'orders' },
    { label: 'Market', to: '/customer/market', icon: 'market' },
    { label: 'Chat history', to: '/customer/chat-history', icon: 'sparkle' },
    { label: 'Planning', to: '/customer/planning', icon: 'trend' },
    { label: 'Basket', to: '/customer/basket', icon: 'basket' },
    { label: 'Suppliers', to: '/customer/suppliers', icon: 'suppliers' },
    { label: 'Payments', to: '/customer/payments', icon: 'payments' },
    { label: 'Warehouses', to: '/customer/warehouses', icon: 'warehouses' },
    { label: 'Cabinet', to: '/customer/cabinet', icon: 'cabinet' }
  ],
  supplier: [
    { label: 'Dashboard', to: '/supplier/dashboard', icon: 'dashboard' },
    { label: 'Orders', to: '/supplier/orders', icon: 'orders' },
    { label: 'Payments', to: '/supplier/payments', icon: 'payments' },
    { label: 'Products', to: '/supplier/products', icon: 'products' },
    { label: 'Chat history', to: '/supplier/chat-history', icon: 'sparkle' },
    { label: 'Delivery', to: '/supplier/delivery', icon: 'truck' },
    { label: 'Warehouses', to: '/supplier/warehouses', icon: 'warehouses' },
    { label: 'Cabinet', to: '/supplier/cabinet', icon: 'cabinet' }
  ]
}

const routes = [
  { path: '/', redirect: () => homeRoute() },
  { path: '/login', name: 'login', component: Login, meta: { public: true, title: 'Sign in' } },
  { path: '/register', name: 'register', component: Register, meta: { public: true, title: 'Register' } },
  { path: '/waiting', name: 'waiting', component: Waiting, meta: { title: 'Approval pending' } },

  {
    path: '/admin',
    component: CabinetLayout,
    meta: { role: 'admin', title: 'Admin' },
    children: [
      { path: '', redirect: { name: 'admin-dashboard' } },
      { path: 'dashboard', name: 'admin-dashboard', component: AdminDashboard, meta: { title: 'Dashboard' } },
      { path: 'suppliers', name: 'admin-suppliers', component: AdminSuppliers, meta: { title: 'Suppliers' } },
      { path: 'customers', name: 'admin-customers', component: AdminCustomers, meta: { title: 'Customers' } },
      { path: 'orders', name: 'admin-orders', component: AdminOrders, meta: { title: 'Orders' } },
      { path: 'chat-history', name: 'admin-chat-history', component: ChatHistory, meta: { title: 'Chat history' } },
      { path: 'preview', redirect: { name: 'admin-preview-supplier' } },
      { path: 'preview/supplier', name: 'admin-preview-supplier', component: AdminPreviewSupplier, meta: { title: 'Supplier preview' } },
      { path: 'preview/customer', name: 'admin-preview-customer', component: AdminPreviewCustomer, meta: { title: 'Customer preview' } },
      { path: 'settings', redirect: { name: 'admin-categories' } },
      { path: 'settings/categories', name: 'admin-categories', component: AdminCategories, meta: { title: 'Product categories' } },
      { path: 'settings/products', name: 'admin-products', component: AdminProducts, meta: { title: 'Products' } },
      { path: 'settings/regions', name: 'admin-regions', component: AdminRegions, meta: { title: 'Regions' } }
    ]
  },

  {
    path: '/customer',
    component: CabinetLayout,
    meta: { role: 'customer', title: 'Customer' },
    children: [
      { path: '', redirect: { name: 'customer-dashboard' } },
      { path: 'dashboard', name: 'customer-dashboard', component: CustomerDashboard, meta: { title: 'Dashboard' } },
      { path: 'orders', name: 'customer-orders', component: CustomerOrders, meta: { title: 'Orders' } },
      { path: 'market', name: 'customer-market', component: CustomerMarket, meta: { title: 'Market' } },
      { path: 'chat-history', name: 'customer-chat-history', component: ChatHistory, meta: { title: 'Chat history' } },
      { path: 'market/:id', name: 'customer-product-detail', component: CustomerProductDetail, meta: { title: 'Product details' } },
      { path: 'planning', name: 'customer-planning', component: CustomerPlanning, meta: { title: 'Planning' } },
      { path: 'basket', name: 'customer-basket', component: CustomerBasket, meta: { title: 'Basket' } },
      { path: 'suppliers', name: 'customer-suppliers', component: CustomerSuppliers, meta: { title: 'Suppliers' } },
      { path: 'payments', name: 'customer-payments', component: CustomerPayments, meta: { title: 'Payments' } },
      { path: 'warehouses', name: 'customer-warehouses', component: CustomerWarehouses, meta: { title: 'Warehouses' } },
      { path: 'cabinet', name: 'customer-cabinet', component: CustomerCabinet, meta: { title: 'Cabinet' } }
    ]
  },

  {
    path: '/supplier',
    component: CabinetLayout,
    meta: { role: 'supplier', title: 'Supplier' },
    children: [
      { path: '', redirect: { name: 'supplier-dashboard' } },
      { path: 'dashboard', name: 'supplier-dashboard', component: SupplierDashboard, meta: { title: 'Dashboard' } },
      { path: 'orders', name: 'supplier-orders', component: SupplierOrders, meta: { title: 'Orders' } },
      { path: 'payments', name: 'supplier-payments', component: SupplierPayments, meta: { title: 'Payments' } },
      { path: 'products', name: 'supplier-products', component: SupplierProducts, meta: { title: 'Products' } },
      { path: 'chat-history', name: 'supplier-chat-history', component: ChatHistory, meta: { title: 'Chat history' } },
      { path: 'delivery', name: 'supplier-delivery', component: SupplierDelivery, meta: { title: 'Delivery' } },
      { path: 'warehouses', name: 'supplier-warehouses', component: SupplierWarehouses, meta: { title: 'Warehouses' } },
      { path: 'cabinet', name: 'supplier-cabinet', component: SupplierCabinet, meta: { title: 'Cabinet' } }
    ]
  },

  { path: '/:pathMatch(.*)*', redirect: () => homeRoute() }
]

function homeRoute() {
  const auth = useAuthStore()
  if (!auth.isAuthenticated) return '/login'
  if (!auth.isAdmin && !auth.isConfirmed) return '/waiting'
  if (auth.role && NAV_CONFIG[auth.role]) return `/${auth.role}`
  return '/login'
}

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  }
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (to.meta.public) {
    return true
  }

  if (!auth.isAuthenticated) {
    return { name: 'login' }
  }

  // Restore the user from the token after a full page reload.
  if (!auth.user) {
    try {
      await auth.me()
    } catch {
      auth.forceLogout()
      return { name: 'login' }
    }
  }

  if (to.name === 'waiting') {
    return true
  }

  if (!auth.isAdmin && !auth.isConfirmed) {
    return { name: 'waiting' }
  }

  const requiredRole = to.meta.role
  if (requiredRole && auth.role !== requiredRole) {
    return homeRoute()
  }

  return true
})

export default router

import { Navigate, Route, Routes } from 'react-router-dom'
import { Toaster } from 'sonner'
import { AppLayout } from '@/components/layout/AppLayout'
import { RequireAuth } from '@/components/layout/RequireAuth'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
import { Containers } from '@/pages/Containers'
import { ContainerDetail } from '@/pages/ContainerDetail'
import { Images } from '@/pages/Images'
import { Volumes } from '@/pages/Volumes'
import { Networks } from '@/pages/Networks'
import { Compose } from '@/pages/Compose'
import { Events } from '@/pages/Events'
import { Settings } from '@/pages/Settings'
import { NotFound } from '@/pages/NotFound'
import { useTheme } from '@/hooks/useTheme'

/** 路由表（SPEC §5.2） */
export function App() {
  const { resolvedTheme } = useTheme()
  return (
    <>
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <RequireAuth>
            <AppLayout />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Dashboard />} />
        <Route path="/events" element={<Events />} />
        <Route path="/containers" element={<Containers />} />
        <Route path="/containers/:id" element={<ContainerDetail />} />
        <Route path="/images" element={<Images />} />
        <Route path="/volumes" element={<Volumes />} />
        <Route path="/networks" element={<Networks />} />
        <Route path="/compose" element={<Compose />} />
        <Route path="/compose/:project" element={<Compose />} />
        <Route path="/settings" element={<Settings />} />
      </Route>
      <Route path="/404" element={<NotFound />} />
      <Route path="*" element={<Navigate to="/404" replace />} />
    </Routes>
    <Toaster
      position="bottom-right"
      theme={resolvedTheme}
      toastOptions={{
        style: {
          background: 'rgb(var(--bg-card))',
          border: '1px solid rgb(var(--border))',
          color: 'rgb(var(--fg-primary))',
        },
      }}
    />
    </>
  )
}
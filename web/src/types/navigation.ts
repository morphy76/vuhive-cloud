import { LucideIcon, LayoutDashboard, Layers, PlayCircle, CalendarClock, Sliders, BookOpen } from 'lucide-react'

export type RouteId = 'dashboard' | 'suites' | 'runs' | 'schedules' | 'profiles' | 'cookbook'

export interface NavItem {
  id: RouteId
  label: string
  icon: LucideIcon
  description: string
}

export const PRIMARY_NAV_ITEMS: NavItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: LayoutDashboard,
    description: 'Control plane overview and cluster metrics',
  },
  {
    id: 'suites',
    label: 'Suites',
    icon: Layers,
    description: 'Test suite definitions and build artifacts',
  },
  {
    id: 'runs',
    label: 'Runs',
    icon: PlayCircle,
    description: 'Live executions, status and performance KPIs',
  },
  {
    id: 'schedules',
    label: 'Schedules',
    icon: CalendarClock,
    description: 'Native Kubernetes CronJob schedules',
  },
  {
    id: 'profiles',
    label: 'Profiles',
    icon: Sliders,
    description: 'Kubernetes compute resources, affinities, and tolerations',
  },
  {
    id: 'cookbook',
    label: 'Cookbook',
    icon: BookOpen,
    description: 'Control plane adoption guide and API recipes',
  },
]

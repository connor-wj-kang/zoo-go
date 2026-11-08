import {ClipboardList, Home, ListChecks, User, Wallet, Search} from 'lucide-react'
import {NavLink} from 'react-router-dom'
import {cn} from '@/utils/cn'

export const TABS = {
    owner: [
        {to: '/owner', label: '首页', icon: Home, end: true},
        {to: '/owner/orders', label: '订单', icon: ClipboardList},
        {to: '/owner/me', label: '我的', icon: User},
    ],
    student: [
        {to: '/student', label: '任务大厅', icon: Search, end: true},
        {to: '/student/my', label: '我的任务', icon: ListChecks},
        {to: '/student/earnings', label: '收益', icon: Wallet},
        {to: '/student/verify', label: '我的', icon: User},
    ],
}

export default function TabBar({role}) {
    return (
        <nav className="flex shrink-0 border-t border-border bg-white pb-5 pt-1.5">
            {TABS[role].map(({to, label, icon: Icon, end}) => (
                <NavLink
                    key={to}
                    to={to}
                    end={end}
                    className={({isActive}) =>
                        cn(
                            'flex flex-1 flex-col items-center gap-0.5 py-1 text-[11px] transition-colors',
                            isActive ? 'font-semibold text-primary-600' : 'text-muted-foreground',
                        )
                    }
                >
                    <Icon size={22} strokeWidth={2} />
                    {label}
                </NavLink>
            ))}
        </nav>
    )
}

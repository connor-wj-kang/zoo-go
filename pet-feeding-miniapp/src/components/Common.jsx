import {BadgeCheck, Star, X} from 'lucide-react'
import {STATUS_META} from '@/data/mock'
import {cn} from '@/utils/cn'

const TONES = {
    primary: 'bg-primary-50 text-primary-700',
    accent: 'bg-accent-100 text-accent-700',
    secondary: 'bg-secondary-50 text-secondary-700',
    neutral: 'bg-gray-100 text-gray-500',
}

export function StatusTag({status}) {
    const meta = STATUS_META[status] || STATUS_META.EXPIRED
    return (
        <span className={cn('rounded-full px-2.5 py-0.5 text-[11px] font-semibold', TONES[meta.tone])}>
            {meta.label}
        </span>
    )
}

export function Verified({className}) {
    return (
        <span className={cn('inline-flex items-center gap-0.5 text-[11px] font-medium text-secondary-600', className)}>
            <BadgeCheck size={13} />
            已核验
        </span>
    )
}

export function Rating({value, done}) {
    return (
        <span className="inline-flex items-center gap-0.5 text-[11px] text-muted-foreground">
            <Star size={12} className="fill-accent-400 text-accent-400" />
            <b className="font-semibold text-foreground">{value}</b>
            {done != null && <span>· {done}单</span>}
        </span>
    )
}

export function Sheet({open, onClose, title, children}) {
    if (!open) {
        return null
    }
    return (
        <div className="fixed inset-0 z-50 flex items-end">
            <div className="absolute inset-0 animate-fade-in bg-black/40" onClick={onClose} />
            <div className="relative max-h-[85%] w-full animate-sheet-up overflow-y-auto rounded-t-3xl bg-white p-5 pb-8">
                <div className="mb-4 flex items-center justify-between">
                    <h3 className="text-base font-bold">{title}</h3>
                    <button type="button" onClick={onClose} className="rounded-full bg-muted p-1.5" aria-label="关闭">
                        <X size={16} />
                    </button>
                </div>
                {children}
            </div>
        </div>
    )
}

export function Section({title, extra, children, className}) {
    return (
        <section className={cn('mt-5', className)}>
            <div className="mb-2.5 flex items-center justify-between px-1">
                <h2 className="text-[15px] font-bold">{title}</h2>
                {extra}
            </div>
            {children}
        </section>
    )
}

export function Row({label, children}) {
    return (
        <div className="flex items-start justify-between gap-4 py-2 text-[13px]">
            <span className="shrink-0 text-muted-foreground">{label}</span>
            <span className="text-right font-medium">{children}</span>
        </div>
    )
}

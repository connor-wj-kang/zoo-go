import {useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {ChevronRight, Clock} from 'lucide-react'
import NavBar from '@/components/NavBar'
import ChipGroup from '@/components/ChipGroup'
import {calcStudentGet} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {cn} from '@/utils/cn'

const TONE = {
    primary: 'bg-primary-50 text-primary-700',
    accent: 'bg-accent-100 text-accent-700',
    secondary: 'bg-secondary-50 text-secondary-700',
    neutral: 'bg-gray-100 text-gray-500',
}

function view(t, mine) {
    const won = t.winnerBidId === mine.id
    if (t.winnerBidId && !won) return {group: '已结束', label: '未中选', tone: 'neutral', to: `/student/task/${t.id}`}
    if (won) {
        if (t.status === 'PENDING_PAYMENT') return {group: '竞价中', label: '中选 · 待主人付款', tone: 'accent', to: `/student/task/${t.id}`}
        if (t.status === 'BOOKED') return {group: '已预约', label: '已预约 · 去履约', tone: 'secondary', to: `/student/fulfill/${t.id}`}
        if (t.status === 'IN_SERVICE') return {group: '进行中', label: '服务中', tone: 'secondary', to: `/student/fulfill/${t.id}`}
        if (t.status === 'PENDING_CONFIRM') return {group: '进行中', label: '待主人验收', tone: 'accent', to: `/student/fulfill/${t.id}`}
        return {group: '已完成', label: '待结算', tone: 'neutral', to: `/student/fulfill/${t.id}`}
    }
    if (t.status === 'CANCELLED') return {group: '已结束', label: '已取消', tone: 'neutral', to: `/student/task/${t.id}`}
    return {group: '竞价中', label: `竞价中 · 报价 ¥${mine.price}`, tone: 'primary', to: `/student/task/${t.id}`}
}

export default function StudentMy() {
    const navigate = useNavigate()
    const {tasks} = useApp()
    const [f, setF] = useState('全部')
    const rows = tasks
        .map((t) => ({t, mine: t.bids.find((b) => b.studentId === 's0')}))
        .filter((r) => r.mine)
        .map((r) => ({...r, v: view(r.t, r.mine)}))
        .filter((r) => f === '全部' || r.v.group === f)

    return (
        <div>
            <NavBar back={false} title="我的任务" />
            <div className="px-4 pt-3">
                <ChipGroup options={['全部', '竞价中', '已预约', '进行中', '已完成']} value={f} onChange={setF} />
                <div className="mt-4 space-y-3">
                    {rows.map(({t, mine, v}) => {
                        const pet = petOf(t)
                        return (
                            <button key={t.id} type="button" onClick={() => navigate(v.to)} className="card flex w-full gap-3 text-left transition active:scale-[0.98]">
                                <img src={pet.img} alt={`${pet.name} 照片`} className="h-16 w-16 shrink-0 rounded-2xl object-cover" />
                                <div className="min-w-0 flex-1">
                                    <div className="flex items-center justify-between">
                                        <b className="text-[15px]">{pet.name} · {t.services[0]}等{t.services.length}项</b>
                                        <ChevronRight size={16} className="text-muted-foreground" />
                                    </div>
                                    <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground"><Clock size={12} />{t.day} {t.window}</p>
                                    <div className="mt-2 flex items-center justify-between">
                                        <span className={cn('rounded-full px-2.5 py-0.5 text-[11px] font-semibold', TONE[v.tone])}>{v.label}</span>
                                        <span className="text-xs text-muted-foreground">预计实收 <b className="text-secondary-600">¥{calcStudentGet(t.price ?? mine.price)}</b></span>
                                    </div>
                                </div>
                            </button>
                        )
                    })}
                    {!rows.length && <p className="py-16 text-center text-sm text-muted-foreground">暂无任务，去大厅看看吧</p>}
                </div>
            </div>
        </div>
    )
}

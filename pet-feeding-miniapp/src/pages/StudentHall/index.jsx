import {useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {BadgeCheck, Bell, Clock, Flame, MapPin, Navigation, Zap} from 'lucide-react'
import NavBar from '@/components/NavBar'
import ChipGroup from '@/components/ChipGroup'
import {calcStudentGet} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {fmtTime, useCountdown} from '@/utils/useCountdown'

function HallCard({task, onOpen}) {
    const pet = petOf(task)
    const left = useCountdown(task.remain)
    const mine = task.bids.find((b) => b.studentId === 's0')
    const dist = task.dist ?? 0.5
    return (
        <button type="button" onClick={() => onOpen(task)} className="card w-full text-left transition active:scale-[0.98]">
            <div className="flex gap-3">
                <img src={pet.img} alt={`${pet.name} 照片`} className="h-20 w-20 shrink-0 rounded-2xl object-cover" />
                <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between">
                        <b className="text-[15px]">{pet.name} · {pet.desc}</b>
                        {task.mode === 'fixed' ? (
                            <span className="flex items-center gap-0.5 rounded-full bg-accent-100 px-2 py-0.5 text-[11px] font-semibold text-accent-700"><Zap size={11} />一口价</span>
                        ) : (
                            <span className="rounded-full bg-primary-50 px-2 py-0.5 text-[11px] font-semibold text-primary-700">竞价</span>
                        )}
                    </div>
                    <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground"><Clock size={12} />{task.day} {task.window}</p>
                    <p className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground"><MapPin size={12} />{task.community} · 约 {dist}km</p>
                    <p className="mt-1 truncate text-xs">{task.services.join(' · ')}</p>
                </div>
            </div>
            <div className="mt-3 flex items-end justify-between border-t border-border/60 pt-3">
                <div>
                    <span className="text-xs text-muted-foreground">{task.mode === 'fixed' ? '成交价' : '预算上限'}</span>
                    <b className="ml-1 text-xl text-primary-600">¥{task.budget}</b>
                    <p className="text-[11px] text-muted-foreground">最高实收约 ¥{calcStudentGet(task.budget)}</p>
                </div>
                <div className="text-right">
                    {mine && <span className="mb-1 block text-[11px] font-semibold text-secondary-600">已报价 ¥{mine.price}</span>}
                    <span className="flex items-center justify-end gap-1 text-xs text-primary-600"><Flame size={13} />截止 {fmtTime(left)}</span>
                </div>
            </div>
        </button>
    )
}

export default function StudentHall() {
    const navigate = useNavigate()
    const {tasks, toast} = useApp()
    const [sort, setSort] = useState('距离最近')
    const [filter, setFilter] = useState('全部')

    const list = tasks
        .filter((t) => t.status === 'BIDDING')
        .filter((t) => filter === '全部' || (filter === '一口价' ? t.mode === 'fixed' : t.mode === 'bid'))
        .sort((a, b) => (sort === '距离最近' ? (a.dist ?? 0.5) - (b.dist ?? 0.5) : b.budget - a.budget))

    return (
        <div>
            <NavBar
                back={false}
                title="任务大厅"
                right={<button type="button" onClick={() => toast('暂无新消息')} aria-label="消息"><Bell size={20} /></button>}
            />
            <div className="px-4 pt-2">
                <div className="flex items-center gap-2 rounded-2xl bg-secondary-50 p-3 text-xs text-secondary-800">
                    <BadgeCheck size={18} className="shrink-0 text-secondary-600" />
                    <span>你已通过实名+在校认证，可在 <b>3km</b> 范围内报价。成交前仅显示模糊位置。</span>
                </div>
                <div className="mt-3 flex items-center justify-between">
                    <ChipGroup options={['全部', '竞价', '一口价']} value={filter} onChange={setFilter} />
                </div>
                <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <Navigation size={12} />排序：
                    <ChipGroup options={['距离最近', '预算最高']} value={sort} onChange={setSort} />
                </div>
                <div className="mt-4 space-y-3">
                    {list.map((t) => <HallCard key={t.id} task={t} onOpen={(x) => navigate(`/student/task/${x.id}`)} />)}
                    {!list.length && <p className="py-16 text-center text-sm text-muted-foreground">附近暂无任务</p>}
                </div>
            </div>
        </div>
    )
}

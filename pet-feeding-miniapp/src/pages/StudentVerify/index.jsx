import {useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {BadgeCheck, CalendarClock, ChevronRight, GraduationCap, IdCard, Repeat, Star, UserCheck} from 'lucide-react'
import NavBar from '@/components/NavBar'
import ChipGroup from '@/components/ChipGroup'
import {Section} from '@/components/Common'
import {STUDENTS, TIME_WINDOWS} from '@/data/mock'
import {useApp} from '@/store/AppContext'

const CHECKS = [
    [IdCard, '实名认证', '已通过'],
    [UserCheck, '年满 18 周岁', '已确认'],
    [GraduationCap, '在校状态核验', '已通过 · 有效期至 2027-06-30'],
]

export default function StudentVerify() {
    const navigate = useNavigate()
    const {toast} = useApp()
    const me = STUDENTS.s0
    const [radius, setRadius] = useState('3km')
    const [slots, setSlots] = useState([TIME_WINDOWS[1], TIME_WINDOWS[3]])

    return (
        <div>
            <NavBar back={false} title="我的" />
            <div className="space-y-4 px-4 pt-3">
                <div className="card flex items-center gap-4 bg-gradient-to-br from-secondary-50 to-white">
                    <img src={me.avatar} alt="学生头像" className="h-16 w-16 rounded-full border-2 border-white object-cover shadow" />
                    <div className="flex-1">
                        <b className="text-lg">{me.name}</b>
                        <p className="flex items-center gap-1 text-xs text-secondary-600"><BadgeCheck size={14} />学生已认证</p>
                    </div>
                    <div className="text-center">
                        <b className="flex items-center gap-0.5 text-xl text-accent-600"><Star size={16} className="fill-accent-400 text-accent-400" />{me.rating}</b>
                        <p className="text-[11px] text-muted-foreground">完成 {me.done} 单</p>
                    </div>
                </div>

                <Section title="认证状态" className="mt-0">
                    <div className="card divide-y divide-border/60 p-0">
                        {CHECKS.map(([Icon, t, d]) => (
                            <div key={t} className="flex items-center gap-3 p-3.5">
                                <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-secondary-50 text-secondary-600"><Icon size={18} /></span>
                                <div className="flex-1"><b className="text-[13px]">{t}</b><p className="text-[11px] text-muted-foreground">{d}</p></div>
                                <BadgeCheck size={18} className="text-secondary-500" />
                            </div>
                        ))}
                    </div>
                    <p className="mt-2 px-1 text-[11px] text-muted-foreground">证件与学籍材料仅审核人员可见，不对主人展示。</p>
                </Section>

                <Section title="服务设置" className="mt-0">
                    <div className="card space-y-4">
                        <div>
                            <p className="mb-2 text-[13px] font-semibold">最大接单半径</p>
                            <ChipGroup options={['1km', '3km', '5km']} value={radius} onChange={(v) => { setRadius(v); toast(`接单半径已设为 ${v}`) }} />
                        </div>
                        <div>
                            <p className="mb-2 flex items-center gap-1 text-[13px] font-semibold"><CalendarClock size={14} />空闲时段</p>
                            <ChipGroup options={TIME_WINDOWS} value={slots} onChange={setSlots} multiple />
                        </div>
                    </div>
                </Section>

                <button
                    type="button"
                    onClick={() => navigate('/owner')}
                    className="flex w-full items-center gap-3 rounded-2xl bg-gradient-to-r from-primary-500 to-primary-600 p-4 text-left text-white shadow-lg shadow-primary-500/25 active:scale-[0.98]"
                >
                    <Repeat size={22} />
                    <div className="flex-1"><b>切换为宠物主人身份</b><p className="text-xs text-white/80">发布任务、选择学生上门</p></div>
                    <ChevronRight size={18} />
                </button>
            </div>
        </div>
    )
}

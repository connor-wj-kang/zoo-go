import {useState} from 'react'
import {Navigate, useParams} from 'react-router-dom'
import {AlertTriangle, Camera, Check, KeyRound, MapPin, Navigation, Plus} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {StatusTag} from '@/components/Common'
import {IMG, calcStudentGet} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {cn} from '@/utils/cn'

const STOCK_PHOTOS = [IMG.golden, IMG.cat1, IMG.dog2, IMG.cat2]

function Step({n, title, done, active, children}) {
    return (
        <div className={cn('card transition', !active && !done && 'opacity-60')}>
            <div className="flex items-center gap-3">
                <span className={cn(
                    'flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold',
                    done ? 'bg-secondary-500 text-white' : active ? 'bg-primary-500 text-white' : 'bg-gray-100 text-gray-400',
                )}>
                    {done ? <Check size={14} /> : n}
                </span>
                <b className="text-sm">{title}</b>
            </div>
            {children && <div className="mt-3">{children}</div>}
        </div>
    )
}

export default function StudentFulfill() {
    const {id} = useParams()
    const {tasks, checkIn, complete, toast} = useApp()
    const task = tasks.find((t) => t.id === id)
    const [checked, setChecked] = useState([])
    const [photos, setPhotos] = useState([])

    const winner = task?.bids.find((b) => b.id === task.winnerBidId)
    if (!task || !winner || winner.studentId !== 's0') {
        return <Navigate to="/student/my" replace />
    }
    const pet = petOf(task)
    const st = task.status
    const checkedIn = ['IN_SERVICE', 'PENDING_CONFIRM', 'COMPLETED'].includes(st)
    const finished = ['PENDING_CONFIRM', 'COMPLETED'].includes(st)
    const ready = checked.length === task.services.length && photos.length >= 2
    const shownPhotos = finished ? task.photos : photos

    const finish = () => {
        complete(task.id, photos)
        toast('已提交完工，等待主人验收')
    }

    return (
        <div>
            <NavBar title="上门履约" backTo="/student/my" />
            <div className="space-y-3 px-4 pt-2 pb-32">
                <div className="card flex items-center gap-3">
                    <img src={pet.img} alt={`${pet.name} 照片`} className="h-14 w-14 rounded-2xl object-cover" />
                    <div className="flex-1">
                        <b>{pet.name} · {task.services.length} 项照护</b>
                        <p className="text-xs text-muted-foreground">{task.day} {task.window}</p>
                    </div>
                    <StatusTag status={st} />
                </div>

                {['BOOKED', 'IN_SERVICE', 'PENDING_CONFIRM', 'COMPLETED'].includes(st) && (
                    <div className="rounded-2xl bg-secondary-50 p-3.5 text-xs text-secondary-800">
                        <p className="flex items-center gap-1.5 font-semibold"><MapPin size={14} />3 号楼 2 单元 401（马连洼）</p>
                        <p className="mt-1 flex items-center gap-1.5"><KeyRound size={13} />钥匙：到场与主人交接，禁止在聊天中发送门锁密码</p>
                    </div>
                )}

                <Step n={1} title="到场签到" done={checkedIn} active={st === 'BOOKED'}>
                    {st === 'BOOKED' ? (
                        <button type="button" onClick={() => { checkIn(task.id); toast('签到成功，位置校验通过') }} className="btn-primary w-full">
                            <Navigation size={16} />到场签到
                        </button>
                    ) : <p className="text-xs text-secondary-600">已签到 · 位置合理性校验通过</p>}
                </Step>

                <Step n={2} title="照护清单（逐项勾选）" done={finished || checked.length === task.services.length} active={st === 'IN_SERVICE'}>
                    <div className="space-y-2">
                        {task.services.map((s) => {
                            const on = finished || checked.includes(s)
                            return (
                                <button
                                    key={s}
                                    type="button"
                                    disabled={st !== 'IN_SERVICE'}
                                    onClick={() => setChecked((c) => (c.includes(s) ? c.filter((x) => x !== s) : [...c, s]))}
                                    className={cn('flex w-full items-center gap-2.5 rounded-xl border p-2.5 text-left text-sm', on ? 'border-secondary-300 bg-secondary-50' : 'border-border')}
                                >
                                    <span className={cn('flex h-5 w-5 items-center justify-center rounded-md border', on ? 'border-secondary-500 bg-secondary-500 text-white' : 'border-gray-300')}>
                                        {on && <Check size={13} />}
                                    </span>
                                    {s}
                                </button>
                            )
                        })}
                    </div>
                </Step>

                <Step n={3} title="上传服务照片（至少 2 张）" done={finished || photos.length >= 2} active={st === 'IN_SERVICE'}>
                    <div className="grid grid-cols-4 gap-2">
                        {shownPhotos.map((p, i) => <img key={`${p}-${i}`} src={p} alt={`凭证 ${i + 1}`} className="aspect-square rounded-xl object-cover" />)}
                        {st === 'IN_SERVICE' && photos.length < 4 && (
                            <button
                                type="button"
                                onClick={() => setPhotos((l) => [...l, STOCK_PHOTOS[l.length % STOCK_PHOTOS.length]])}
                                className="flex aspect-square flex-col items-center justify-center gap-0.5 rounded-xl border border-dashed border-primary-300 bg-primary-50/50 text-[10px] text-primary-600"
                            >
                                <Camera size={18} /><Plus size={10} />拍照
                            </button>
                        )}
                    </div>
                    <p className="mt-2 text-[11px] text-muted-foreground">请勿拍到他人面部、证件与密码。</p>
                </Step>

                {st === 'PENDING_CONFIRM' && <p className="card text-center text-sm text-accent-700">已提交完工，等待主人验收（24 小时内确认，否则按规则自动完成）</p>}
                {st === 'COMPLETED' && <p className="card text-center text-sm text-secondary-700">订单已完成，预计实收 ¥{calcStudentGet(task.price)} 进入待结算</p>}

                <button type="button" onClick={() => toast('已通知客服并暂停结算')} className="mx-auto flex items-center gap-1 pt-2 text-xs text-muted-foreground underline">
                    <AlertTriangle size={13} />遇到异常？一键上报
                </button>
            </div>

            {st === 'IN_SERVICE' && (
                <div className="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-white p-4 pb-6">
                    <button type="button" disabled={!ready} onClick={finish} className="btn-primary w-full">
                        {ready ? '结束服务并提交' : `还需完成清单 ${task.services.length - checked.length} 项 / 照片 ${Math.max(0, 2 - photos.length)} 张`}
                    </button>
                </div>
            )}
        </div>
    )
}

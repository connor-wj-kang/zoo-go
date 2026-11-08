import {useNavigate} from 'react-router-dom'
import {ChevronRight, Clock, Users} from 'lucide-react'
import {StatusTag} from '@/components/Common'
import {petOf} from '@/store/AppContext'

export const ownerPath = (t) =>
    t.status === 'BIDDING' || t.status === 'PENDING_PAYMENT' ? `/owner/task/${t.id}` : `/owner/order/${t.id}`

export default function OwnerTaskCard({task}) {
    const navigate = useNavigate()
    const pet = petOf(task)
    return (
        <button
            type="button"
            onClick={() => navigate(ownerPath(task))}
            className="card flex w-full gap-3 text-left transition active:scale-[0.98]"
        >
            <img src={pet.img} alt={`${pet.name} 宠物照片`} className="h-16 w-16 shrink-0 rounded-2xl object-cover" />
            <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between">
                    <b className="text-[15px]">
                        {pet.name} · {task.services[0]}等{task.services.length}项
                    </b>
                    <StatusTag status={task.status} />
                </div>
                <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock size={12} />
                    {task.day} {task.window}
                </p>
                <div className="mt-2 flex items-center justify-between">
                    {task.price ? (
                        <span className="text-sm font-bold text-primary-600">成交 ¥{task.price}</span>
                    ) : (
                        <span className="text-sm font-bold text-primary-600">预算 ≤ ¥{task.budget}</span>
                    )}
                    {task.status === 'BIDDING' ? (
                        <span className="flex items-center gap-1 text-xs text-secondary-600">
                            <Users size={12} />
                            {task.bids.length} 人已报价
                        </span>
                    ) : (
                        <ChevronRight size={16} className="text-muted-foreground" />
                    )}
                </div>
            </div>
        </button>
    )
}

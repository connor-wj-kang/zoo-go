import {useState} from 'react'
import NavBar from '@/components/NavBar'
import OwnerTaskCard from '@/components/OwnerTaskCard'
import ChipGroup from '@/components/ChipGroup'
import {useApp} from '@/store/AppContext'

const FILTERS = {
    全部: () => true,
    竞价中: (t) => ['BIDDING', 'PENDING_PAYMENT'].includes(t.status),
    履约中: (t) => ['BOOKED', 'IN_SERVICE', 'PENDING_CONFIRM'].includes(t.status),
    已完成: (t) => ['COMPLETED', 'SETTLED'].includes(t.status),
}

export default function OwnerOrders() {
    const {tasks} = useApp()
    const [f, setF] = useState('全部')
    const list = tasks.filter((t) => t.ownerId === 'o0' && FILTERS[f](t))
    return (
        <div>
            <NavBar back={false} title="我的订单" />
            <div className="px-4 pt-3">
                <ChipGroup options={Object.keys(FILTERS)} value={f} onChange={setF} />
                <div className="mt-4 space-y-3">
                    {list.map((t) => <OwnerTaskCard key={t.id} task={t} />)}
                    {!list.length && <p className="py-16 text-center text-sm text-muted-foreground">暂无订单</p>}
                </div>
            </div>
        </div>
    )
}

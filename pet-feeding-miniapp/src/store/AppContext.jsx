import {createContext, useCallback, useContext, useMemo, useState} from 'react'
import {PETS, SEED_TASKS} from '@/data/mock'

const AppContext = createContext(null)

export const petOf = (task) => {
    const p = PETS.find((x) => x.id === task.petId)
    if (p) {
        return {name: p.name, img: p.img, desc: p.breed, species: p.species}
    }
    return {name: task.petName, img: task.petImg, desc: task.petDesc, species: ''}
}

export function AppProvider({children}) {
    const [tasks, setTasks] = useState(SEED_TASKS)
    const [toastMsg, setToastMsg] = useState('')

    const toast = useCallback((msg) => {
        setToastMsg(msg)
        window.clearTimeout(window.__toastTimer)
        window.__toastTimer = window.setTimeout(() => setToastMsg(''), 1800)
    }, [])

    const patch = useCallback((id, fn) => {
        setTasks((list) => list.map((t) => (t.id === id ? {...t, ...fn(t)} : t)))
    }, [])

    const value = useMemo(() => ({
        tasks,
        toastMsg,
        toast,
        addTask(data) {
            const id = `t${Date.now()}`
            setTasks((list) => [
                {
                    id, ownerId: 'o0', status: 'BIDDING', candidate: null, winnerBidId: null,
                    photos: [], rated: false, bids: [], ...data,
                },
                ...list,
            ])
            return id
        },
        selectBid(taskId, bidId) {
            patch(taskId, (t) => ({
                status: 'PENDING_PAYMENT',
                winnerBidId: bidId,
                price: t.bids.find((b) => b.id === bidId).price,
                payRemain: 600,
            }))
        },
        reopen(taskId) {
            patch(taskId, () => ({status: 'BIDDING', winnerBidId: null, price: undefined, remain: 1800}))
        },
        pay: (taskId) => patch(taskId, () => ({status: 'BOOKED'})),
        cancel: (taskId) => patch(taskId, () => ({status: 'CANCELLED'})),
        checkIn: (taskId) => patch(taskId, () => ({status: 'IN_SERVICE'})),
        complete: (taskId, photos) => patch(taskId, () => ({status: 'PENDING_CONFIRM', photos})),
        confirm: (taskId) => patch(taskId, () => ({status: 'COMPLETED'})),
        rate: (taskId) => patch(taskId, () => ({rated: true})),
        submitBid(taskId, price, msg) {
            patch(taskId, (t) => {
                const mine = t.bids.find((b) => b.studentId === 's0')
                if (mine) {
                    return {bids: t.bids.map((b) => (b.studentId === 's0' ? {...b, price, msg, rev: b.rev + 1} : b))}
                }
                return {bids: [...t.bids, {id: `bs-${taskId}`, studentId: 's0', price, msg, rev: 0}]}
            })
        },
        withdrawBid: (taskId) => patch(taskId, (t) => ({bids: t.bids.filter((b) => b.studentId !== 's0')})),
        applyFixed: (taskId) => patch(taskId, () => ({candidate: 's0'})),
    }), [tasks, toastMsg, toast, patch])

    return <AppContext.Provider value={value}>{children}</AppContext.Provider>
}

export const useApp = () => useContext(AppContext)

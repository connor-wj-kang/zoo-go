import {useEffect, useState} from 'react'

export function fmtTime(sec) {
    const s = Math.max(0, sec)
    const m = String(Math.floor(s / 60)).padStart(2, '0')
    const r = String(s % 60).padStart(2, '0')
    return `${m}:${r}`
}

export function useCountdown(initial) {
    const [left, setLeft] = useState(initial)
    useEffect(() => {
        setLeft(initial)
    }, [initial])
    useEffect(() => {
        if (left <= 0) {
            return undefined
        }
        const timer = window.setTimeout(() => setLeft((v) => v - 1), 1000)
        return () => window.clearTimeout(timer)
    }, [left])
    return left
}

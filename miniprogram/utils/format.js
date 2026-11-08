const {mediaUrl} = require('./api')

const STATUS_META = {
    BIDDING: {text: '竞价中', tone: 'primary'},
    PENDING_PAYMENT: {text: '待付款', tone: 'accent'},
    BOOKED: {text: '已预约', tone: 'secondary'},
    IN_SERVICE: {text: '服务中', tone: 'secondary'},
    PENDING_CONFIRM: {text: '待验收', tone: 'accent'},
    REFUNDING: {text: '退款中', tone: 'accent'},
    COMPLETED: {text: '已完成', tone: 'neutral'},
    SETTLED: {text: '已结算', tone: 'neutral'},
    REFUNDED: {text: '已退款', tone: 'neutral'},
    CANCELLED: {text: '已取消', tone: 'neutral'},
    EXPIRED: {text: '已过期', tone: 'neutral'},
    PAID: {text: '已支付', tone: 'secondary'},
    PAID_LATE: {text: '超时支付待退款', tone: 'accent'},
    PREPARING: {text: '支付准备中', tone: 'accent'},
    PREPARED: {text: '待微信支付', tone: 'accent'},
    CLOSED: {text: '支付已关闭', tone: 'neutral'},
    REQUESTED: {text: '退款待审核', tone: 'accent'},
    PROCESSING: {text: '退款处理中', tone: 'accent'},
    SUCCESS: {text: '退款成功', tone: 'secondary'},
    FAILED: {text: '处理失败', tone: 'neutral'},
    REJECTED: {text: '退款已驳回', tone: 'neutral'},
    UNPAID: {text: '待支付', tone: 'accent'},
}

const money = (fen) => {
    const value = Number(fen || 0) / 100
    return Number.isInteger(value) ? String(value) : value.toFixed(2)
}

const fmtTime = (seconds) => {
    const total = Math.max(0, Number(seconds) || 0)
    const minutes = Math.floor(total / 60)
    const secs = Math.floor(total % 60)
    return `${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
}

const decorateTask = (task) => {
    const meta = STATUS_META[task.status] || {text: task.status, tone: 'neutral'}
    const refundMeta = task.refund ? (STATUS_META[task.refund.status] || {text: task.refund.status, tone: 'neutral'}) : null
    const services = task.services || []
    return {
        ...task,
        pet: {
            ...(task.pet || {}),
            image: mediaUrl(task.pet && task.pet.image),
        },
        photos: (task.photos || []).map(mediaUrl),
        statusText: meta.text,
        statusClass: `status-${meta.tone}`,
        budgetText: money(task.budgetFen),
        priceText: money(task.priceFen),
        servicesText: services.join(' · '),
        servicesInline: services.join('、'),
        serviceTitle: services.length ? `${services[0]}等${services.length}项` : '上门照护',
        distanceText: task.distanceKm !== undefined && task.distanceKm !== null ? Number(task.distanceKm).toFixed(1) : '',
        remainingText: fmtTime(task.remainingSec),
        payRemainingText: fmtTime(task.payRemainingSec),
        winnerName: task.winner ? task.winner.name : '',
        winnerAvatar: task.winner ? task.winner.avatar : '',
        orderStatusText: task.order ? (STATUS_META[task.order.status] || {text: task.order.status}).text : '',
        paymentStatusText: task.order && task.order.paymentStatus ? (STATUS_META[task.order.paymentStatus] || {text: task.order.paymentStatus}).text : '',
        amountText: task.order ? money(task.order.amountFen) : money(task.priceFen),
        feeText: task.order ? money(task.order.feeFen) : '',
        studentReceiveText: task.order ? money(task.order.studentFen) : '',
        hasOrder: Boolean(task.order),
        refundStatusText: refundMeta ? refundMeta.text : '',
        refundStatusClass: refundMeta ? `status-${refundMeta.tone}` : '',
        refundAmountText: task.refund ? money(task.refund.amountFen) : '',
        canRefund: task.status === 'BOOKED' && task.order && task.order.status === 'PAID' && !task.refund,
        canPay: task.status === 'PENDING_PAYMENT',
        isFulfilling: ['BOOKED', 'IN_SERVICE', 'PENDING_CONFIRM'].includes(task.status),
        isDone: ['COMPLETED', 'SETTLED'].includes(task.status),
    }
}

const decorateBid = (bid) => ({
    ...bid,
    student: {
        ...bid.student,
        avatar: mediaUrl(bid.student.avatar),
    },
    priceText: money(bid.priceFen),
    distanceText: Number(bid.student.distanceKm || 0).toFixed(1),
    ratingText: Number(bid.student.rating || 0).toFixed(1),
    avatar: mediaUrl(bid.student.avatar),
})

const tapFeedback = () => {
    if (wx.vibrateShort) {
        wx.vibrateShort({type: 'light'})
    }
}

const relativeTime = (value) => {
    const time = new Date(value).getTime()
    if (!Number.isFinite(time)) {
        return ''
    }
    const seconds = Math.max(0, Math.floor((Date.now() - time) / 1000))
    if (seconds < 60) {
        return '刚刚'
    }
    if (seconds < 3600) {
        return `${Math.floor(seconds / 60)} 分钟前`
    }
    if (seconds < 86400) {
        return `${Math.floor(seconds / 3600)} 小时前`
    }
    if (seconds < 7 * 86400) {
        return `${Math.floor(seconds / 86400)} 天前`
    }
    const date = new Date(value)
    return `${date.getMonth() + 1}月${date.getDate()}日`
}

const decoratePost = (post) => {
    const author = post.author || {}
    return {
        ...post,
        author: {
            ...author,
            avatarUrl: mediaUrl(author.avatar),
            initial: (author.name || '宠').slice(0, 1),
            verified: Boolean(author.ownerVerified || author.studentVerified),
        },
        images: (post.images || []).map((path) => ({path, url: mediaUrl(path)})),
        timeText: relativeTime(post.createdAt),
    }
}

module.exports = {
    STATUS_META,
    money,
    fmtTime,
    decorateTask,
    decorateBid,
    decoratePost,
    relativeTime,
    tapFeedback,
}

const OWNER_TABS = [
    {label: '首页', icon: '⌂', url: '/pages/owner/home/index', key: 'home'},
    {label: '社区', icon: '◎', url: '/pages/community/index?role=owner', key: 'community'},
    {label: '订单', icon: '▤', url: '/pages/owner/orders/index', key: 'orders'},
    {label: '我的', icon: '○', url: '/pages/owner/me/index', key: 'me'},
]

const STUDENT_TABS = [
    {label: '任务大厅', icon: '⌕', url: '/pages/student/hall/index', key: 'hall'},
    {label: '社区', icon: '◎', url: '/pages/community/index?role=student', key: 'community'},
    {label: '我的任务', icon: '✓', url: '/pages/student/tasks/index', key: 'tasks'},
    {label: '收益', icon: '¥', url: '/pages/student/earnings/index', key: 'earnings'},
    {label: '我的', icon: '○', url: '/pages/student/verify/index', key: 'verify'},
]

Component({
    properties: {
        role: {type: String, value: 'owner'},
        active: {type: String, value: 'home'},
    },
    data: {tabs: OWNER_TABS},
    observers: {
        role(value) {
            this.setData({tabs: value === 'student' ? STUDENT_TABS : OWNER_TABS})
        },
    },
    lifetimes: {
        attached() {
            this.setData({tabs: this.data.role === 'student' ? STUDENT_TABS : OWNER_TABS})
        },
    },
    methods: {
        onTap(event) {
            const {url, key} = event.currentTarget.dataset
            if (key === this.data.active) {
                return
            }
            wx.reLaunch({url})
        },
    },
})

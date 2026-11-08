const {request} = require('../../../utils/api')
const {decorateTask, money} = require('../../../utils/format')

const FILTERS = [
    {value: 'active', label: '进行中'},
    {value: 'completed', label: '已完成'},
    {value: 'all', label: '全部'},
]

Page({
    data: {
        loading: true,
        allTasks: [],
        tasks: [],
        filter: 'active',
        filters: FILTERS,
    },

    onShow() {
        this.loadTasks()
    },

    onPullDownRefresh() {
        this.loadTasks().finally(() => wx.stopPullDownRefresh())
    },

    async loadTasks() {
        try {
            const tasks = await request('/v1/student/tasks')
            const allTasks = (tasks || []).map((task) => {
                const view = decorateTask(task)
                return {
                    ...view,
                    myBidText: task.myBid ? money(task.myBid.priceFen) : '',
                    myBidStatusText: task.myBid && task.myBid.status === 'WON' ? '已中选' : task.myBid ? '等待选择' : '',
                }
            })
            this.setData({loading: false, allTasks})
            this.applyFilter(this.data.filter, allTasks)
        } catch (error) {
            this.setData({loading: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },

    selectFilter(event) {
        const filter = event.currentTarget.dataset.value
        this.setData({filter})
        this.applyFilter(filter, this.data.allTasks)
    },

    applyFilter(filter, allTasks) {
        let tasks = allTasks
        if (filter === 'active') {
            tasks = allTasks.filter((task) => !['COMPLETED', 'SETTLED', 'CANCELLED', 'EXPIRED'].includes(task.status))
        } else if (filter === 'completed') {
            tasks = allTasks.filter((task) => ['COMPLETED', 'SETTLED'].includes(task.status))
        }
        this.setData({tasks})
    },

    openTask(event) {
        const task = this.data.allTasks.find((item) => item.id === event.currentTarget.dataset.id)
        if (!task) {
            return
        }
        if (task.orderId && !['BIDDING', 'PENDING_PAYMENT'].includes(task.status)) {
            wx.navigateTo({url: `/pages/student/fulfill/index?id=${task.orderId}`})
        } else {
            wx.navigateTo({url: `/pages/student/task/index?id=${task.id}`})
        }
    },

    goHall() {
        wx.reLaunch({url: '/pages/student/hall/index'})
    },
})

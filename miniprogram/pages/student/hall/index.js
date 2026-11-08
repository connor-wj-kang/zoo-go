const {request} = require('../../../utils/api')
const {decorateTask, money} = require('../../../utils/format')

Page({
    data: {
        loading: true,
        error: '',
        student: null,
        tasks: [],
        availableCount: 0,
        locating: false,
        locationDenied: false,
    },

    onLoad() {
        this.loadHall()
    },

    onShow() {
        if (!this.data.loading) {
            this.loadHall(false)
        }
    },

    onPullDownRefresh() {
        this.loadHall(false).finally(() => wx.stopPullDownRefresh())
    },

    async loadHall(showLoading = true) {
        if (showLoading) {
            this.setData({loading: true, error: '', locationDenied: false})
        }
        this.setData({locating: true})
        try {
            const location = await this.getCurrentLocation()
            await request('/v1/student/location', {
                method: 'POST',
                data: {
                    latitude: location.latitude,
                    longitude: location.longitude,
                    radiusKm: 5,
                },
            })
            const [student, tasks] = await Promise.all([
                request('/v1/student/profile'),
                request('/v1/student/tasks/available'),
            ])
            const decorated = (tasks || []).map((task) => {
                const view = decorateTask(task)
                return {
                    ...view,
                    suggestedText: money(Math.max(2000, Math.round(task.budgetFen * 0.8))),
                    fixedMode: task.mode === 'fixed',
                    myBidText: task.myBid ? money(task.myBid.priceFen) : '',
                }
            })
            this.setData({
                loading: false,
                error: '',
                student,
                tasks: decorated,
                availableCount: decorated.length,
                locating: false,
                locationDenied: false,
            })
        } catch (error) {
            const locationDenied = /auth deny|auth denied|authorize no response|location/i.test(error.errMsg || '')
            this.setData({
                loading: false,
                locating: false,
                locationDenied,
                error: locationDenied ? '需要获取你的当前位置，才能匹配 5 公里内的任务。' : error.message,
            })
        }
    },

    getCurrentLocation() {
        return new Promise((resolve, reject) => {
            wx.getLocation({
                type: 'gcj02',
                isHighAccuracy: true,
                highAccuracyExpireTime: 4000,
                success: resolve,
                fail: reject,
            })
        })
    },

    goTask(event) {
        wx.navigateTo({url: `/pages/student/task/index?id=${event.currentTarget.dataset.id}`})
    },

    goOwner() {
        wx.reLaunch({url: '/pages/owner/home/index'})
    },

    goVerify() {
        wx.reLaunch({url: '/pages/student/verify/index'})
    },

    retry() {
        this.loadHall()
    },

    openLocationSetting() {
        wx.openSetting({
            success: (result) => {
                if (result.authSetting['scope.userLocation']) {
                    this.loadHall()
                }
            },
        })
    },
})

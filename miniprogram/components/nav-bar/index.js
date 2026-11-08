Component({
    properties: {
        title: {type: String, value: ''},
        back: {type: Boolean, value: false},
        rightIcon: {type: String, value: ''},
        rightText: {type: String, value: ''},
        fallbackUrl: {type: String, value: '/pages/owner/home/index'},
    },
    data: {statusBarHeight: 20},
    lifetimes: {
        attached() {
            const info = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()
            this.setData({statusBarHeight: info.statusBarHeight || 20})
        },
    },
    methods: {
        onBack() {
            const pages = getCurrentPages()
            if (pages.length > 1) {
                wx.navigateBack()
            } else {
                wx.reLaunch({url: this.data.fallbackUrl || '/pages/owner/home/index'})
            }
        },
        onRight() {
            this.triggerEvent('right')
        },
    },
})

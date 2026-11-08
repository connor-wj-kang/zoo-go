const {request, upload, mediaUrl} = require('../../utils/api')
const {decoratePost, tapFeedback} = require('../../utils/format')

Page({
    data: {
        role: 'owner',
        fallbackUrl: '/pages/owner/home/index',
        loading: true,
        loadingMore: false,
        publishing: false,
        error: '',
        posts: [],
        nextCursor: '',
        hasMore: false,
        content: '',
        images: [],
        canPublish: false,
    },

    onLoad(options) {
        const role = options.role === 'student' ? 'student' : 'owner'
        this.setData({
            role,
            fallbackUrl: role === 'student' ? '/pages/student/hall/index' : '/pages/owner/home/index',
        })
        this.loadPosts(true)
    },

    onShow() {
        if (!this.data.loading) {
            this.loadPosts(true, false)
        }
    },

    onPullDownRefresh() {
        this.loadPosts(true, false).finally(() => wx.stopPullDownRefresh())
    },

    onReachBottom() {
        this.loadMore()
    },

    async loadPosts(reset = false, showLoading = true) {
        if (reset && showLoading) {
            this.setData({loading: true, error: ''})
        }
        try {
            const cursor = reset ? '' : this.data.nextCursor
            const query = cursor ? `?limit=10&cursor=${encodeURIComponent(cursor)}` : '?limit=10'
            const feed = await request(`/v1/community/posts${query}`)
            const items = (feed.items || []).map(decoratePost)
            this.setData({
                loading: false,
                error: '',
                posts: reset ? items : this.data.posts.concat(items),
                nextCursor: feed.nextCursor || '',
                hasMore: Boolean(feed.hasMore),
            })
        } catch (error) {
            this.setData({loading: false, error: error.message})
        }
    },

    async loadMore() {
        if (this.data.loading || this.data.loadingMore || !this.data.hasMore) {
            return
        }
        this.setData({loadingMore: true})
        await this.loadPosts(false, false)
        this.setData({loadingMore: false})
    },

    onContentInput(event) {
        const content = event.detail.value
        this.setData({
            content,
            canPublish: Boolean(content.trim() || this.data.images.length),
        })
    },

    chooseImages() {
        const remaining = 9 - this.data.images.length
        if (remaining <= 0) {
            wx.showToast({title: '每条动态最多 9 张图片', icon: 'none'})
            return
        }
        if (!wx.chooseMedia && !wx.chooseImage) {
            wx.showToast({title: '当前微信版本不支持选择图片', icon: 'none'})
            return
        }
        const options = {
            count: remaining,
            mediaType: ['image'],
            sourceType: ['album', 'camera'],
            sizeType: ['compressed'],
            success: (result) => {
                const paths = result.tempFiles
                    ? result.tempFiles.map((file) => file.tempFilePath)
                    : (result.tempFilePaths || [])
                const images = this.data.images.concat(paths.filter(Boolean).map((path) => ({path, url: '', src: path})))
                this.setData({
                    images,
                    canPublish: Boolean(this.data.content.trim() || images.length),
                })
            },
        }
        if (wx.chooseMedia) {
            wx.chooseMedia(options)
        } else {
            wx.chooseImage(options)
        }
    },

    removeImage(event) {
        const index = Number(event.currentTarget.dataset.index)
        const images = this.data.images.slice()
        images.splice(index, 1)
        this.setData({
            images,
            canPublish: Boolean(this.data.content.trim() || images.length),
        })
    },

    previewSelectedImage(event) {
        const urls = this.data.images.map((item) => item.src)
        wx.previewImage({
            current: event.currentTarget.dataset.src,
            urls,
        })
    },

    previewPostImage(event) {
        const post = this.data.posts[Number(event.currentTarget.dataset.postIndex)]
        if (!post) {
            return
        }
        wx.previewImage({
            current: event.currentTarget.dataset.src,
            urls: post.images.map((item) => item.url),
        })
    },

    async publish() {
        if (this.data.publishing) {
            return
        }
        const content = this.data.content.trim()
        if (!content && !this.data.images.length) {
            wx.showToast({title: '写点内容或选一张图片', icon: 'none'})
            return
        }
        this.setData({publishing: true})
        try {
            const images = this.data.images.slice()
            for (let index = 0; index < images.length; index += 1) {
                if (!images[index].url) {
                    const result = await upload(images[index].path)
                    images[index] = {
                        ...images[index],
                        url: result.url,
                        src: mediaUrl(result.url),
                    }
                    this.setData({images})
                }
            }
            await request('/v1/community/posts', {
                method: 'POST',
                data: {
                    content,
                    images: images.map((item) => item.url).filter(Boolean),
                },
            })
            tapFeedback()
            this.setData({content: '', images: [], canPublish: false})
            wx.showToast({title: '动态已发布', icon: 'success'})
            await this.loadPosts(true, false)
        } catch (error) {
            wx.showToast({title: error.message, icon: 'none'})
        } finally {
            this.setData({publishing: false})
        }
    },

    retry() {
        this.loadPosts(true)
    },
})

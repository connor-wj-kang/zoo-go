const {request, upload} = require('../../../utils/api')
const {tapFeedback} = require('../../../utils/format')

const FLAGS = ['温顺', '怕生', '爱互动', '会护食', '需喂药', '老年宠物']

Page({
    data: {
        submitting: false,
        name: '',
        species: '猫',
        speciesOptions: [
            {value: '猫', selected: true},
            {value: '狗', selected: false},
        ],
        breed: '',
        age: '',
        note: '',
        imagePath: '',
        flagOptions: FLAGS.map((value) => ({value, selected: false})),
    },

    onNameInput(event) {
        this.setData({name: event.detail.value})
    },

    selectSpecies(event) {
        const value = event.currentTarget.dataset.value
        this.setData({
            species: value,
            speciesOptions: this.data.speciesOptions.map((item) => ({...item, selected: item.value === value})),
        })
    },

    onBreedInput(event) {
        this.setData({breed: event.detail.value})
    },

    onAgeInput(event) {
        this.setData({age: event.detail.value})
    },

    onNoteInput(event) {
        this.setData({note: event.detail.value})
    },

    toggleFlag(event) {
        const value = event.currentTarget.dataset.value
        this.setData({
            flagOptions: this.data.flagOptions.map((item) => item.value === value ? {...item, selected: !item.selected} : item),
        })
    },

    choosePhoto() {
        if (!wx.chooseMedia && !wx.chooseImage) {
            wx.showToast({title: '当前微信版本不支持选择图片', icon: 'none'})
            return
        }
        const options = {
            count: 1,
            mediaType: ['image'],
            sourceType: ['album', 'camera'],
            sizeType: ['compressed'],
            success: (result) => {
                const file = result.tempFiles ? result.tempFiles[0] : null
                const path = file ? file.tempFilePath : (result.tempFilePaths || [])[0]
                if (path) {
                    this.setData({imagePath: path})
                }
            },
        }
        if (wx.chooseMedia) {
            wx.chooseMedia(options)
        } else {
            wx.chooseImage(options)
        }
    },

    async submit() {
        if (this.data.submitting) {
            return
        }
        const name = this.data.name.trim()
        if (!name) {
            wx.showToast({title: '请填写宠物名字', icon: 'none'})
            return
        }
        this.setData({submitting: true})
        try {
            let image = ''
            if (this.data.imagePath) {
                image = (await upload(this.data.imagePath)).url
            }
            await request('/v1/pets', {
                method: 'POST',
                data: {
                    name,
                    species: this.data.species,
                    breed: this.data.breed,
                    age: this.data.age,
                    image,
                    note: this.data.note,
                    flags: this.data.flagOptions.filter((item) => item.selected).map((item) => item.value),
                },
            })
            tapFeedback()
            wx.showToast({title: '宠物档案已创建', icon: 'success'})
            setTimeout(() => {
                const pages = getCurrentPages()
                if (pages.length > 1) {
                    wx.navigateBack()
                } else {
                    wx.reLaunch({url: '/pages/owner/home/index'})
                }
            }, 500)
        } catch (error) {
            this.setData({submitting: false})
            wx.showToast({title: error.message, icon: 'none'})
        }
    },
})

const U = (id, w = 600) => `https://images.unsplash.com/${id}?w=${w}&q=70&auto=format&fit=crop`

export const IMG = {
  golden: U('photo-1552053831-71594a27632d'),
  cat1: U('photo-1514888286974-6c03e2ca1dba'),
  dog2: U('photo-1543466835-00a7907e9de1'),
  cat2: U('photo-1573865526739-10659fec78a5'),
  corgi: U('photo-1587300003388-59208cc962cb'),
  dog3: U('photo-1533738363-b7f9aef128ce'),
  hero: U('photo-1450778869180-41d0601e046e', 900),
}

export const AVATAR = {
  s0: U('photo-1507003211169-0a1dd7228f2d', 200),
  s1: U('photo-1494790108377-be9c29b29330', 200),
  s2: U('photo-1500648767791-00dcc994a43e', 200),
  s3: U('photo-1438761681033-6461ffad8d80', 200),
}

export const STUDENTS = {
  s0: {id: 's0', name: '小林同学', avatar: AVATAR.s0, rating: 4.9, done: 21, dist: 0.7, intro: '养猫三年，熟悉猫咪应激表现，可准时上门。'},
  s1: {id: 's1', name: '林小夏', avatar: AVATAR.s1, rating: 4.9, done: 38, dist: 0.6, intro: '本地在校生，有养犬经验，每次服务附详细照片。'},
  s2: {id: 's2', name: '陈子墨', avatar: AVATAR.s2, rating: 4.8, done: 26, dist: 1.2, intro: '周末时间充裕，擅长大型犬喂养与清理。'},
  s3: {id: 's3', name: '王可心', avatar: AVATAR.s3, rating: 5.0, done: 12, dist: 0.9, intro: '爱猫人士，耐心细致，可按清单逐项完成。'},
}

export const PETS = [
  {id: 'p1', name: '豆豆', species: '狗', breed: '金毛 · 3岁', img: IMG.golden, note: '性格温顺，每餐 1 勺半狗粮，需换水', flags: ['温顺', '无攻击性']},
  {id: 'p2', name: '咪咪', species: '猫', breed: '英短 · 2岁', img: IMG.cat1, note: '怕生，进门后动作轻，先清理猫砂再喂食', flags: ['怕生', '需轻声']},
]

export const SERVICE_TYPES = ['喂食', '换水', '清理猫砂', '清理生活区', '拍照反馈']
export const TIME_WINDOWS = ['09:00-10:00', '12:00-13:00', '15:00-16:00', '18:00-19:00', '20:00-21:00']
export const DAYS = ['11-05 周四', '11-06 周五', '11-07 周六', '11-08 周日', '11-09 周一']
export const DEADLINES = [
  {label: '15分钟', sec: 900},
  {label: '30分钟', sec: 1800},
  {label: '60分钟', sec: 3600},
  {label: '2小时', sec: 7200},
]

export const COMMUNITIES = ['马连洼·朗润园附近', '马连洼·农大南路附近', '马连洼·圆明园西路附近']

export const STATUS_META = {
  BIDDING: {label: '竞价中', tone: 'primary'},
  PENDING_PAYMENT: {label: '待付款', tone: 'accent'},
  BOOKED: {label: '已预约', tone: 'secondary'},
  IN_SERVICE: {label: '服务中', tone: 'secondary'},
  PENDING_CONFIRM: {label: '待验收', tone: 'accent'},
  COMPLETED: {label: '已完成', tone: 'neutral'},
  SETTLED: {label: '已结算', tone: 'neutral'},
  EXPIRED: {label: '已过期', tone: 'neutral'},
  CANCELLED: {label: '已取消', tone: 'neutral'},
}

export const FEE_RATE = 0.1
export const calcStudentGet = (price) => Math.round(price * (1 - FEE_RATE) * 100) / 100

const base = {candidate: null, winnerBidId: null, photos: [], rated: false}

export const SEED_TASKS = [
  {
    ...base, id: 't1', ownerId: 'o0', petId: 'p1', mode: 'bid', status: 'BIDDING', budget: 60,
    day: '11-07 周六', window: '09:00-10:00', services: ['喂食', '换水', '拍照反馈'],
    community: COMMUNITIES[0], remain: 2538, note: '周末回老家，请按每餐 1 勺半喂食并换水，拍 2 张照片即可。',
    bids: [
      {id: 'b1', studentId: 's1', price: 48, msg: '本小区附近，可提前 10 分钟到。', rev: 0},
      {id: 'b2', studentId: 's3', price: 42, msg: '有养犬经验，会按清单逐项完成。', rev: 1},
      {id: 'b3', studentId: 's2', price: 52, msg: '大型犬经验丰富，附完成照片。', rev: 0},
    ],
  },
  {
    ...base, id: 't2', ownerId: 'o0', petId: 'p2', mode: 'bid', status: 'BOOKED', budget: 50,
    day: '11-05 周四', window: '18:00-19:00', services: ['喂食', '清理猫砂', '拍照反馈'],
    community: COMMUNITIES[0], remain: 0, note: '咪咪怕生，请轻声进门，先清理猫砂再喂食。', winnerBidId: 'b9', price: 45,
    bids: [{id: 'b9', studentId: 's0', price: 45, msg: '养猫三年，会放轻动作。', rev: 0}],
  },
  {
    ...base, id: 't3', ownerId: 'o0', petId: 'p1', mode: 'bid', status: 'COMPLETED', budget: 50,
    day: '10-28 周三', window: '12:00-13:00', services: ['喂食', '换水'],
    community: COMMUNITIES[0], remain: 0, note: '', winnerBidId: 'b8', price: 40, rated: true,
    photos: [IMG.golden, IMG.dog2],
    bids: [{id: 'b8', studentId: 's1', price: 40, msg: '', rev: 0}],
  },
  {
    ...base, id: 'h1', ownerId: 'o1', petName: '布丁', petImg: IMG.corgi, petDesc: '柯基 · 2岁', mode: 'bid',
    status: 'BIDDING', budget: 50, day: '11-06 周五', window: '12:00-13:00', services: ['喂食', '换水'],
    community: COMMUNITIES[1], dist: 0.8, remain: 1820, note: '食欲好，需防止偷吃，喂完请收好狗盆。', bids: [],
  },
  {
    ...base, id: 'h2', ownerId: 'o2', petName: '团团', petImg: IMG.cat2, petDesc: '布偶 · 1岁', mode: 'fixed',
    status: 'BIDDING', budget: 38, day: '11-06 周五', window: '18:00-19:00', services: ['喂食', '清理猫砂'],
    community: COMMUNITIES[2], dist: 1.5, remain: 3100, note: '一口价 38 元，需主人确认后付款。', bids: [],
  },
  {
    ...base, id: 'h3', ownerId: 'o3', petName: '球球', petImg: IMG.dog3, petDesc: '泰迪 · 4岁', mode: 'bid',
    status: 'BIDDING', budget: 70, day: '11-08 周日', window: '09:00-10:00', services: ['喂食', '换水', '清理生活区'],
    community: COMMUNITIES[1], dist: 2.3, remain: 5400, note: '两只狗狗，较活泼，进门时注意别让它们出门。', bids: [],
  },
]

export default [
  {
    title: 'Anasayfa',
    to: { name: 'root' },
    icon: { icon: 'tabler-smart-home' },
  },
  {
    title: 'Kullanıcı Yönetimi',
    to: { name: 'users' },
    icon: { icon: 'tabler-users' },
    role: ['admin'],
  },
]

import moment from 'moment'

// TODO: defaulta gerek yok.
export default function tarihFormat(tarih: moment.MomentInput) {
  return moment(tarih).locale('tr').format('DD.MM.YYYY') // TODO: tr ve . yı neden ekledin
}

export function tarihSaatFormat(tarih: moment.MomentInput) {
  return moment(tarih).format('DD/MM/YYYY HH:mm')
}

import Swal from 'sweetalert2'
import 'sweetalert2/src/sweetalert2.scss'
import { staticPrimaryColor } from '@/plugins/vuetify/theme'

export function ErrorPopup(text: string) {
  if (text === '-')
    return

  return Swal.fire({
    html: text,
    icon: 'error',
    confirmButtonText: 'Tamam',
    confirmButtonColor: staticPrimaryColor,
    customClass: {
      container: 'error-popup-container',
    },
    didOpen: () => {
      const container = document.querySelector<HTMLElement>('.error-popup-container')
      if (container)
        container.style.zIndex = '9999'
    },
  })
}

export function SuccessPopup(text: string) {
  return Swal.fire({
    html: text,
    icon: 'success',
    confirmButtonText: 'Tamam',
    confirmButtonColor: staticPrimaryColor,
    customClass: {
      container: 'success-popup-container',
    },
    didOpen: () => {
      const container = document.querySelector<HTMLElement>('.success-popup-container')
      if (container)
        container.style.zIndex = '9999'
    },
  })
}

export function WarningPopup(
  html: string,
  confimText?: string,
  cancelText?: string,
) {
  confimText = confimText || 'Tamam'

  return Swal.fire({
    title: 'Dikkat',
    html,
    icon: 'warning',
    showCancelButton: !!cancelText,
    confirmButtonText: confimText,
    confirmButtonColor: staticPrimaryColor,
    cancelButtonText: cancelText,
    customClass: {
      container: 'warning-popup-container',
    },
    didOpen: () => {
      const container = document.querySelector<HTMLElement>('.warning-popup-container')
      if (container)
        container.style.zIndex = '9999'
    },
  })
}

export async function ConfirmPopup(
  text: string,
  confimText: string,
  cancelText: string,
) {
  const result = await Swal.fire({
    title: text,
    showCancelButton: true,
    showConfirmButton: true,
    confirmButtonText: confimText,
    confirmButtonColor: staticPrimaryColor,
    cancelButtonText: cancelText,
    customClass: {
      container: 'confirm-popup-container',
    },
    didOpen: () => {
      const container = document.querySelector<HTMLElement>('.confirm-popup-container')
      if (container)
        container.style.zIndex = '9999'
    },
  })

  return result.isConfirmed
}
export function SuccessToast(text = 'İşlem Başarılı') {
  const Toast = Swal.mixin({
    toast: true,
    position: 'top-end',
    showConfirmButton: false,
    timer: 3000,
    timerProgressBar: true,
  })

  return Toast.fire({
    icon: 'success',
    title: text,
  })
}

export function ErrorToast(text: string) {
  const Toast = Swal.mixin({
    toast: true,
    position: 'top-end',
    showConfirmButton: false,
    timer: 3000,
    timerProgressBar: true,
  })

  return Toast.fire({
    icon: 'error',
    title: text,
  })
}

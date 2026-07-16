<template>
  <!--
    Hesap sayfaları inert ve sahte veri gösteriyor (spec §2.1).

    Bu not, "yalancı değil" sınırının parçası: ekranda "Elif Yılmaz" ve bir
    Kadıköy adresi duruyor. Ziyaretçi bunun kendi hesabı olduğunu sanmamalı —
    tasarım korunuyor ama site sahip olmadığı bir durumu iddia etmiyor.

    Faz 2'de üyelik gelince bu bileşen silinir.
  -->
  <p
    class="rounded border border-outline-variant/40 bg-surface-container-low px-4 py-3 text-center text-xs text-on-surface-variant"
    role="note"
  >
    Bu ekranlar önizleme amaçlıdır; üyelik sistemi çok yakında açılıyor.
    Siparişlerinizi şimdilik WhatsApp üzerinden alıyoruz.
  </p>
</template>

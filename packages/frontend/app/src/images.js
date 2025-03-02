import deleteIconUrl from './assets/images/delete-cross.svg?no-inline';
import processIconUrl from './assets/images/process.svg?no-inline';
import editIconUrl from './assets/images/edit_icon.svg?no-inline';
import ratesIcon from './assets/images/rates.svg?no-inline'
import bucketIcon from './assets/images/bucket.svg?no-inline'
import buildingsIcon from './assets/images/building.svg?no-inline'
import apartmentsIcon from './assets/images/apartments.svg?no-inline'
import nextPageIcon from './assets/images/chevron-double-down.svg?no-inline'
import backBtnIcon from './assets/images/left-arrow.svg?no-inline'
import crossIcon from './assets/images/cross.svg?no-inline'
import selectIcon from './assets/images/selectIcon.svg?no-inline'
import checkBoxIcon from './assets/images/checkbox-icon.svg?no-inline'

window.deleteIconUrl = deleteIconUrl;
window.processIconUrl = processIconUrl;
window.editIconUrl = editIconUrl;
window.nextPageIcon = nextPageIcon;
window.backBtnIcon = backBtnIcon;
window.crossIcon = crossIcon;
window.selectIcon = selectIcon;
window.checkBoxIcon = checkBoxIcon;

window.ratesIcon = ratesIcon;
window.bucketIcon = bucketIcon;
window.buildingsIcon = buildingsIcon;
window.apartmentsIcon = apartmentsIcon;

window.NAV_ICONS = new Map([
  ['nav-apartments', apartmentsIcon],
  ['nav-buildings', buildingsIcon],
  ['nav-receipts', ratesIcon],
  ['nav-rates', ratesIcon],
  ['nav-users', bucketIcon],
  ['nav-bcv-files', bucketIcon],
]);
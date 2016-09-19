package app

import ()

var keys_cautionary = "<strong style='opacity:0.8;'>public key</strong> <small>%v</small><br><strong>private key</strong> <small>%v</small><br><br><small>Remember your passphrase and keep your private key, well.. private!</small>"

//  If you forget your passphrase or your account is compromised you will need your private key to regain access.

var unauthorized = "Unauthorized"

var account_remove_failure = "<span style='color:red;'>Could not remove account<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small></span>"
var account_remove_success = "Removed account<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small>"

var admin_remove_failure = "<span style='color:red;'>Could not remove admin<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small></span>"
var admin_remove_success = "Removed admin<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small>"

var form_already_exists = "<span style='color:red;'>Form already exists</span>"
var submit_form_failure = "<span style='color:red;'>Failed to submit form</span>"
var submit_form_success = "<strong>Form ID</strong> <small>%v</small>"

var find_form_failure = "<span style='color:red;'>Failed to find<br><strong>Form ID</strong> <small>%v</small></span>"

var resolve_form_failure = "<span style='color:red;'>Failed to resolve<br><strong>Form ID</strong> <small>%v</small></span>"
var resolve_form_success = "Resolved<br><strong>Form ID</strong> <small>%v</small>"

var search_forms_failure = "<span style='color:red;'>Failed to find forms</span>"

var find_admin_failure = "<span style='color:red;'>Failed to find admin</span>"

var line = "<strong style='opacity:0.8;'>%v</strong> <small>%v</small>" + "<br>"

var invalid_pubkey_passphrase = "invalid public key + passphrase"

var user_not_found = "user with public key not found"
var user_already_exists = "user with public key already exists"

var admin_not_found = "admin with public key not found"
var admin_already_exists = "admin with public key already exists"
var admin_db_full = "admin db full"
var admin_update = "admin-%v-update"

var select_option = "<option value='%v'>%v</option>"

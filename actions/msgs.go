package actions

const (
	keys_cautionary     = "<strong style='opacity:0.8;'>public key</strong> <small>%v</small><br><strong style='opacity:0.8;'>private key</strong> <small>%v</small><br><br><small>Remember your passphrase and keep your private key, well.. private!</small>"
	create_user_failure = "<span style='color:red;'>Could not create user</span>"

	unauthorized = "<span style='color:red;'>Unauthorized</span>"

	remove_user_failure = "<span style='color:red;'>Could not remove user<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small></span>"
	remove_user_success = "Removed user<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small>"

	remove_admin_failure = "<span style='color:red;'>Could not remove admin<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small></span>"
	remove_admin_success = "Removed admin<br><strong>Public Key</strong> <small>%v</small><br><strong>Passphrase</strong> <small>%v</small>"

	submit_form_failure = "<span style='color:red;'>Failed to submit form</span>"
	submit_form_success = "<strong>Form ID</strong> <small>%v</small>"

	form_already_exists = "<span style='color:red;'>Form already exists</span>"
	find_form_failure   = "<span style='color:red;'>Failed to find<br><strong>Form ID</strong> <small>%v</small></span>"

	resolve_form_failure = "<span style='color:red;'>Failed to resolve<br><strong>Form ID</strong> <small>%v</small></span>"
	resolve_form_success = "Resolved<br><strong>Form ID</strong> <small>%v</small>"

	search_forms_failure = "<span style='color:red;'>Failed to find forms</span>"

	find_admin_failure = "<span style='color:red;'>Failed to find admin</span>"

	line = "<strong style='opacity:0.8;'>%v</strong> <small>%v</small>" + "<br>"

	admin_update = "admin-%v-update"

	select_option = "<option value='%v'>%v</option>"

	calc_metric_success = "<strong style='opacity: 0.9;'>%v</strong><br> <small>%v</small>"
	calc_metric_failure = "<span style='color:red;'>Could not calculate metric -- zero forms found</span>"
)

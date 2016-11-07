package actions

const (
	invalid_public_key  = "<span style='color:red;'>Invalid public key <really-small>%X</really-small></span>"
	invalid_private_key = "<span style='color:red;'>Invalid private key</span>"
	invalid_formID      = "<span style='color:red;'>Invalid form ID <small>%X</small></span>"

	create_account_success = "<strong style='opacity:0.8;'>public key</strong> <really-small>%X</really-small><br><strong style='opacity:0.8;'>private key</strong> <really-small>%X</really-small>"
	create_account_failure = "<span style='color:red;'>Could not create account</span>"

	remove_account_failure = "<span style='color:red;'>Could not remove account with public key <really-small>%X</really-small></span>"
	remove_account_success = "Removed account with public key <really-small>%X</really-small>"

	create_admin_failure = "<span style='color:red;'>Failed to create admin</span>"
	create_admin_success = "Created admin<br><strong style='opacity:0.8;'>public key</strong> <really-small>%X</really-small><br><strong style='opacity:0.8;'>private key</strong> <really-small>%X</really-small>"

	remove_admin_failure = "<span style='color:red;'>Could not remove admin with public key <really-small>%X</really-small></span>"
	remove_admin_success = "Removed admin with public key <really-small>%X</really-small>"

	submit_form_failure = "<span style='color:red;'>Failed to submit form</span>"
	submit_form_success = "Submitted form with ID <small>%X</small>"

	form_already_exists = "<span style='color:red;'>Form already exists</span>"

	find_form_failure   = "<span style='color:red;'>Failed to find form with ID <small>%X</small></span>"
	decode_form_failure = "<span style='color:red;'>Failed to decode form with ID <small>%X</small></span>"

	resolve_form_failure = "<span style='color:red;'>Failed to resolve form with ID <small>%X</small></span>"
	resolve_form_success = "Resolved form with ID <small>%X</small>"

	search_forms_failure = "<span style='color:red;'>Failed to find forms</span>"

	already_connected = "<span style='color:red;'>You are already connected to the network</span>"
	connect_failure   = "<span style='color:red;'>Failed to connect to network</span>"

	find_admin_failure   = "<span style='color:red;'>Could not find admin with public key <small>%v</small></span>"
	send_message_failure = "<span style='color:red;'>Failed to send message to <small>%v</small></span>"
	send_message_success = "<span style='color:red;'>Sent message to <small>%v</small></span>"

	select_option = "<option value='%v'>%v</option>"
)

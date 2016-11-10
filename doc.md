<img src="https://raw.githubusercontent.com/zballs/comit/master/cmd/comit/static/images/logo/logo.png" height="120" width="120" />

comit provides an interface through which constituents
	
- submit issue forms
- send messages to other users
- find an issue form by ID 
- search forms by type, location, date
- view submissions in real time via the feeds

Admins (e.g. government officials, local organizers) can resolve forms but cannot submit forms. Otherwise, admins and constituents have the same abilities.  


## API Endpoint

This is the root endpoint

`http://localhost:8888/`

## Account 

To create a new account or remove an existing account, go to the following endpoint in your web-browser:

`http://localhost:8888/account`

### Create an account 
- enter a `secret` for generating the public and private keys
- click `create` and receive your public and private key
- keeps your keys in a safe place

### Remove an account 
- enter your public and private keys in hexadecimal form
- click `remove` and receive confirmation of account removal or failure message 

## Network 

To connect to the network, view real-time submissions on the feeds, send and check messages, submit and resolve forms, go to the following endpoint in your web browser:

`http://localhost:8888/network`

### Connect to the network 
- enter your public and private keys in hexadecimal form
- click `connect` to proceed or receive failure message
- if credentials are valid, the feed and submit/resolve form area will appear

### View the feeds
- select one or more feeds from the dropdown
- click `update` and watch the submissions pop up!

### Send a message 
- enter the recipient public key in hexadecimal form
- type a message 
- click `send` and receive confirmation of message send or failure message

### Check messages
- click `check` to receive unseen messages

### Submit a form (constituent)
- select an issue type 
- enter a location
- write a description 
- include an image, audio, or video file (optional)
- choose whether you want to submit the form anonymously
- choose whether you want to send an update to the feed
- click `submit` to receive confirmation of submission or failure message

### Resolve a form (admin)
- enter the form ID for a resolved issue
- click `resolve` to change the form status
- if key credentials are invalid or user does not have permission, a failure message will be returned 

## Forms 

To find and search for forms, go to the following endpoint in your web-browser:

`http://localhost:8888/forms` 

### Find a form 
- enter the form ID in hexadecimal form 
- click `find` to receive form information or failure message 

### Search for forms 
- select an issue type 
- select a location (optional)
- enter a date range for submissions (optional)
- click `search` to view matching forms or receive failure message
 





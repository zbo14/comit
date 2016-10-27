# Comit

Comit provides an interface through which constituents
	
- submit issue forms, on or off the network
- find an issue form by ID 
- search forms by type, location, date
- view submissions in real time via the feeds
- send messages to admins

Admins (e.g. government officials, local organizers) can

- resolve issue forms on the network
- find and search forms 
- receive messages from constituents 


## API Endpoint

This is the root endpoint

`http://localhost:8888/`

## Account 

To create a new account, remove an existing account, or connect to the network, go to the following endpoint in your web-browser:

`http://localhost:8888/account`

### Create an account 
- enter a `secret` for generating the public and private keys
- click `create` to receive your public and private key

### Remove an account 
- enter your public and private keys in hexademical form
- click `remove` to receive confirmation of removal or failure message 

### Connect to network 
- enter your public and private keys in hexadecimal form
- click `connect` to receive confirmation of connection or failure message
- If connection is successful, you will be redirected to the network endpoint

## Network 

To view real-time submissions on the feeds or send a message to an admin, connect to the network via the `account` endpoint. You will then be redirected to the following endpoint:

`http://localhost:8888/network`

### View the feeds
- select a feed from the dropdown
- watch the submissions pop-up!

### Send a message
- enter a message into the textarea 
- enter the public key of the recipient in hexadecimal form 
- click `send` to receive confirmation that message was sent or failure message

## Forms 

To submit, find, and search forms, go to the following endpoint in your web-browser:

`http://localhost:8888/forms` 

### Submit a form 
- select an issue type 
- enter a location
- write a description 
- enter public and private keys in hexadecimal form
- click `submit` to receive confirmation of submission or failure message

### Find a form 
- enter the form ID in hexadecimal form 
- click `find` to receive form information or failure message 

### Search forms 
- select an issue type 
- select a location (optional)
- enter a date range for submissions (optional)
- click `search` to view matching forms or failure message
 





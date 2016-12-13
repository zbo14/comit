# Comit

Comit provides an interface through which citizens can
	
- submit issues and receive receipts
- query blocks of issues
- find an issue by form ID 
- search issues by type, location, date of submission
- view new submissions in the feed


## API Endpoint

This is the root endpoint

`http://localhost:8888/`

## Home 

To create a new account or login to an existing account, go to the following endpoint in your web-browser:

`http://localhost:8888/home`

### Create an account 
- enter a `username` for the account and a `password` to generate your keys
- click `create` to receive your public and private key in hexadecimal form
- save your keys in a safe place

### Login
- enter your public and private keys in hexadecimal form
- click `enter` to login 
- if the credentials are valid, you will be redirected to the `citizen` endpoint

## Citizen 

To submit and query issues and view submissions in the feed, login to the homepage and you will be redirected to the following endpoint:

`http://localhost:8888/citizen`

### Update feed
- select one or more feeds from the dropdown
- click `update` to view submissions in real time

### Submit an issue
- select an issue type 
- enter a location
- write a description 
- include an image or video file (optional)
- click `submit` to broadcast the form to the network
- if/when the form is *broadcast* to the network, you will receive a form ID
- if/when the form is *committed* to the blockchain, you will receive a `receipt`

### Query a block 
TODO

### Find an issue 
- enter the form ID in hexadecimal form 
- click `find` to view form content

### Search for issues 
- select an issue type 
- select a location (optional)
- select a range for date of submission (optional)
- click `search` to view content of matching forms
 





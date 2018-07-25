# Monzo Sweeper

[![Build Status](https://travis-ci.org/lildude/monzo-sweeper.svg?branch=master)](https://travis-ci.org/lildude/monzo-sweeper) [![Coverage Status](https://coveralls.io/repos/github/lildude/monzo-sweeper/badge.svg?branch=master)](https://coveralls.io/github/lildude/monzo-sweeper?branch=master)

This is a simple Go application that sweeps the balance of your account at the time of receiving a deposit above a set threshold to a designated savings pot. This is a great way of automating your monthly savings and compliments the coin jar rounding up functionality already provided by Monzo.

This is designed to be run on Heroku as I already use Heroku and this runs quite happily in the free micro dyno. It's probably not that hard to modify it for AWS Lambda or any other FaaS service provider.

## How it works

1. Monzo triggers a webhook on each transaction.
2. This webhook is configured to POST the transaction data to this application running on Heroku.
3. The application checks if the the amount is larger than the configured threshold, and if it is, it'll transfer the balance at the time of receiving the incoming transaction to a nominated pot.

## Installation

### Pre-Requisites

- A [Monzo](https://monzo.com) bank account
- A [Monzo Bank Developer](https://developers.monzo.com) account
- A [Heroku](https://heroku.com) account

### Deploying and Configuring Monzo Sweeper

1. If you don't have any pots configured yet, create one on the Account tab in your Monzo app.
2. Deploy Monzo Sweeper to Heroku: [Snazzy button coming :soon:]
3. Make a note of the application URL, this is the webhook URL you'll need to enter on Monzo when registering the webhook.
4. Browse to https://developers.monzo.com/ and sign in. This will take you to the Monzo API playground. Make a note of the Account ID and Access token values from the top of the page. If you have multiple Monzo accounts, enter `/accounts` in the `GET` box and click "Send" to get a list of accounts and make a note of the ID of the account you wish to use.
5. Obtain the Pot ID for the Pot you wish to sweep your money into by:
  - replacing `/accounts` with `/pots` in the `GET` box and clicking "Send".
  - scroll through the results until you find the pot and make a note of the `id` field. This is your Pot ID.
6. Register a webhook by clicking "Register webhook" of the Monzo playground page and replace the `url` value in the "Request Body" field with the application URL from Heroku and click "Send".
7. Set the following configuration variables, either in the Heroku UI, or using the Heroku CLI:
  - `MONZO_ACCOUNT_ID` - your account ID, taken from step 4.  
  - `MONZO_ACCESS_TOKEN` - used to request transfers to savings pot, taken from step 4.
  - `MONZO_SWEEP_POT_ID` -  the identifier of the target savings pot, taken from step 5.
  - `MONZO_SWEEP_THRESHOLD` - the threshold, in _pence_, for incoming payments to trigger a sweep. If not set, sweeping will not occur.

  ```
  $ heroku config:set \
  MONZO_ACCOUNT_ID="your-account-id" \
  MONZO_PERSONAL_ACCESS_TOKEN="your-personal-access-token" \
  MONZO_SWEEP_POT_ID="your-savings-pot-id \
  MONZO_SWEEP_THRESHOLD="your-threshold-in-pence"
  ```

### Local Development and Testing

- Save your Heroku config vars to a `.env` file: `heroku config:get -s  >.env`. Don't commit this file to your repo unless you really don't like your money.
- Start the application: `heroku local`.
- Send test requests to 0.0.0.0:5000 using something like curl or httpie.

### Contributing

Issues and pull requests are both welcome.

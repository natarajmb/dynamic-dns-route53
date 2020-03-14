# Dynamic DNS utility

Dynamic DNS utility to update DNS provider record on external IP change. This program currently works with AWS Route53 and only updates the `A` record. Intended use is to run the program as unix cronjob.

## How to compile

Checkout the code and run the following command from `src` directory

    go build -o ../build/ddr53

_NOTE: prefix above command with GOOS=linux GOARCH=arm GOARM=5 if you intend generate binaies to run on Raspberry Pi_

## How to run

Copy the `config.yaml` from root of the repo into `build` folder and update the values in `config.yaml` to reflect your environment

    ./ddr53 config.yaml

## Sample crontab configration

Assuming both binary and config files are copied `route53` folder on user `sid` to system that runs cronjobs put below config into crontab using `crontab -e` 

    */5 * * * * cd /home/sid/route53 && ./ddr53 config.yaml >> /home/sid/route53/ddr53.log 2>&1

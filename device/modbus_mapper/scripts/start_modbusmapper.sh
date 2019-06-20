#!/bin/sh
CONFIG_DIR=/opt/src/conf

for VAR in $(env)
do
    if [[ ! -z "$(echo $VAR | grep -E '^CONNECTOR_')" ]]; then
        VAR_NAME=$(echo "$VAR" | sed -r "s/([^=]*)=.*/\1/g")
        echo "$VAR_NAME=$(eval echo \$$VAR_NAME)"
        sed -i "s#{$VAR_NAME}#$(eval echo \$$VAR_NAME)#g" $CONFIG_DIR/conf.json
    fi
done

cd /opt/src
node index.js

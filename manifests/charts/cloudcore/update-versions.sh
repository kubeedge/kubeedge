#!/bin/bash

set -e

NEW_VERSION=$1

if [ -z "$NEW_VERSION" ]; then
    echo "Error: Version argument required"
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION="${NEW_VERSION#v}"

echo "Updating Chart.yaml..."
sed -i "s/^version: .*$/version: ${VERSION}/" Chart.yaml
sed -i "s/^appVersion: .*$/appVersion: ${VERSION}/" Chart.yaml

echo "Updating values.yaml..."
sed -i "s/^appVersion: \".*\"$/appVersion: \"${VERSION}\"/" values.yaml

components=("cloudcore" "iptables-manager" "controller-manager")
for component in "${components[@]}"; do
    echo "Updating ${component} version..."
    sed -i "/repository: \"kubeedge\/${component}\"/,/tag:/ s/tag: \"v.*\"/tag: \"v${VERSION}\"/" values.yaml
done

echo -e "\nVerifying changes:"
echo -e "\nChart.yaml contents:"
cat Chart.yaml
echo -e "\nComponent versions in values.yaml:"
grep -A 1 "repository: \"kubeedge" values.yaml
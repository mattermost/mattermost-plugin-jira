readonly COMMITHASH=`jq -r '.localPackages.mattermost_webapp' package.json`

echo "\n\nInstalling mattermost-webapp from the mattermost repo, using commit hash $COMMITHASH\n"

if [ ! -d mattermost-webapp ]; then
  mkdir mattermost-webapp
fi

cd mattermost-webapp

if [ ! -d .git ]; then
  git init
  git config --local uploadpack.allowReachableSHA1InWant true
  git remote add origin https://github.com/mattermost/mattermost.git
fi

git fetch --depth=1 origin $COMMITHASH
git reset --hard FETCH_HEAD

cd ..
mv mattermost-webapp/webapp .
rm -rf mattermost-webapp
mv webapp mattermost-webapp

echo "After movement"

npm i --save-dev ./mattermost-webapp/channels
npm i --save-dev ./mattermost-webapp/platform/types
npm i --save-dev ./mattermost-webapp/platform/client

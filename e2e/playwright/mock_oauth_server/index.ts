require('dotenv').config();

import {ExpiryAlgorithm, makeOAuthServer} from './mock_oauth_server';

const encodedOAuthToken = process.env.MOCK_OAUTH_ACCESS_TOKEN;
if (!encodedOAuthToken) {
    console.error('Please provide an OAuth access token to use');
    process.exit(0);
}

const authorizeURL = '/authorize';
const tokenURL = '/oauth/token';

const mattermostSiteURL = process.env.MM_SERVICESETTINGS_SITEURL || 'http://localhost:8065';
const pluginId = process.env.MM_PLUGIN_ID || 'jira';

const app = makeOAuthServer({
    authorizeURL,
    tokenURL,
    mattermostSiteURL,
    encodedOAuthToken,
    pluginId,
    expiryAlgorithm: ExpiryAlgorithm.ONE_HOUR,
});

const port = process.env.OAUTH_SERVER_PORT || 8080;
app.listen(port, () => {
    console.log(`Mock OAuth server listening on port ${port}`);
});

import {ExpiryAlgorithm, makeOAuthServer, OAuthToken} from '../mock_oauth_server/mock_oauth_server';

const encodedOAuthToken = process.env.PLUGIN_E2E_MOCK_OAUTH_TOKEN;
if (!encodedOAuthToken) {
    console.error('Please provide an OAuth access token to use via env var PLUGIN_E2E_MOCK_OAUTH_TOKEN');
    process.exit(1);
}

export const runOAuthServer = async () => {
    const mattermostSiteURL = process.env.MM_SERVICESETTINGS_SITEURL || 'http://localhost:8065';
    const pluginId = process.env.MM_PLUGIN_ID || 'jira';

    const authorizeURL = '/authorize';
    const tokenURL = '/oauth/token';

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
};

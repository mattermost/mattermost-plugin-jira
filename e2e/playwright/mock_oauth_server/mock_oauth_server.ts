import express from 'express';

export type OAuthToken = {
    access_token: string;
    token_type: 'bearer';
    refresh_token?: string;
    expiry: string;
}

export enum ExpiryAlgorithm {
    NO_EXPIRY = 'no_expiry',
    ONE_HOUR = 'one_hour',
}

const ONE_HOUR_MILLIS = 60 * 60 * 1000;

export type OAuthServerOptions = {
    mattermostSiteURL: string;
    pluginId: string;
    encodedOAuthToken: string;
    authorizeURL: string;
    tokenURL: string;
    expiryAlgorithm: ExpiryAlgorithm;
    redirectUriQueryParameter?: string;
}

export const makeOAuthServer = ({
    mattermostSiteURL,
    pluginId,
    encodedOAuthToken,
    authorizeURL,
    tokenURL,
    expiryAlgorithm,
    redirectUriQueryParameter,
}: OAuthServerOptions): express.Express => {
    if (!encodedOAuthToken) {
        throw new Error(`MockOAuthServer: Please provide an OAuth access token to use`);
    }

    const originalToken = oauthTokenFromBase64(encodedOAuthToken);

    const app = express();

    app.get(authorizeURL, function (req, res) {
        let redirectUri: string;
        if (redirectUriQueryParameter) {
            redirectUri = req.query[redirectUriQueryParameter] as string;
        } else {
            redirectUri = `${mattermostSiteURL}/plugins/${pluginId}/oauth/complete`;
        }

        const state = req.query.state;
        const fullRedirectURL = `${redirectUri}?code=1234&state=${state}`

        res.redirect(fullRedirectURL);
    });

    app.post(tokenURL, function (req, res) {
        let token: OAuthToken;
        switch (expiryAlgorithm) {
            case ExpiryAlgorithm.NO_EXPIRY:
                token = makeTokenWithNoExpiry(originalToken);
                break;
            case ExpiryAlgorithm.ONE_HOUR:
                token = makeTokenWithExpiry(ONE_HOUR_MILLIS, originalToken);
                break;
            default:
                throw new Error(`MockOAuthServer: Unsupported OAuth token expiry algorithm: ${expiryAlgorithm}`);
        }

        res.json(token);
    });

    return app;
}

const makeTokenWithNoExpiry = (token: OAuthToken): OAuthToken => {
    return {
        access_token: token.access_token,
        token_type: 'bearer',
        expiry: '0001-01-01T00:00:00Z',
    };
}

const makeTokenWithExpiry = (expiry: number, token: OAuthToken): OAuthToken => {
    const newExpiry = new Date(new Date().getTime() + expiry).toISOString();

    return {
        access_token: token.access_token,
        refresh_token: token.refresh_token,
        token_type: 'bearer',
        expiry: newExpiry,
    };
}

const oauthTokenFromBase64 = (base64Token: string): OAuthToken => {
    const decoded = Buffer.from(base64Token, 'base64').toString();
    return JSON.parse(decoded);
}

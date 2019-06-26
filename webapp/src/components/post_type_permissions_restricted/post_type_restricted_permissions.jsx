// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';
import PropTypes from 'prop-types';

export default class PostTypeRestrictedPermissions extends React.PureComponent {
    static propTypes = {

        /**
         * The post to render the message for.
         */
        post: PropTypes.object.isRequired,

        /**
         * Logged in user's theme.
         */
        theme: PropTypes.object.isRequired,

        /**
         * The actions connected to the component in the index
         */
        checkPermissions: PropTypes.func.isRequired,
        removePermissions: PropTypes.func.isRequired,
    };

    async componentDidMount() {
        const hasPermissions = await this.props.checkPermissions(this.props.post.props.issue_key);
        if (hasPermissions) {
            this.props.removePermissions(this.props.post);
        }
    }

    render() {
        const post = this.props.post;
        const linkClasses = 'theme markdown__link';
        return (
            <p>
                <a
                    className={linkClasses}
                    href={post.props.issue_link}
                >
                    {post.props.issue_link}
                </a>
                {post.props.placeholder_suffix}
            </p>
        );
    }
}

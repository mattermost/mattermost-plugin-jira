// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

export default class CreateIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool,
        locale: PropTypes.string,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
    };

    static defaultTypes = {
        locale: 'en',
    };

    getLocalizedTitle = () => {
        const {locale} = this.props;
        switch (locale) {
        case 'es':
            return 'Crear incidencia en Jira';
        default:
            return 'Create Jira Issue';
        }
    };

    handleClick = (e) => {
        const {open, postId} = this.props;
        e.preventDefault();
        console.log('opening modal for ', postId);
        open(postId);
    };

    render() {
        if (this.props.isSystemMessage) {
            return null;
        }

        return (
            <li
                role='presentation'
            >
                <button
                    className='style--none'
                    role='menuitem'
                    onClick={this.handleClick}
                >
                    {this.getLocalizedTitle()}
                </button>
            </li>
        );
    }
}

const getStyle = (theme) => ({
    configuration: {
        padding: '1em',
        color: theme.centerChannelBg,
        backgroundColor: theme.centerChannelColor,
    },
});

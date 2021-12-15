// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {injectIntl} from 'react-intl';

export class FormButton extends PureComponent {
    static propTypes = {
        executing: PropTypes.bool,
        disabled: PropTypes.bool,
        executingMessage: PropTypes.node,
        defaultMessage: PropTypes.node,
        btnClass: PropTypes.string,
        extraClasses: PropTypes.string,
        saving: PropTypes.bool,
        savingMessage: PropTypes.string,
        type: PropTypes.string,
        intl: PropTypes.object,
    };

    static defaultProps = {
        disabled: false,
        savingMessage: 'Creating',
        defaultMessage: 'Create',
        btnClass: 'btn-primary',
        extraClasses: '',
    };

    render() {
        const {formatMessage} = this.props.intl;
        const {saving, disabled, savingMessage, defaultMessage, btnClass, extraClasses, ...props} = this.props;

        const message = defaultMessage || formatMessage({defaultMessage: 'Create'});
        const saveMessage = savingMessage || formatMessage({defaultMessage: 'Creating'});

        let contents;
        if (saving) {
            contents = (
                <span>
                    <span
                        className='fa fa-spin fa-spinner'
                        title={'Loading Icon'}
                    />
                    {saveMessage}
                </span>
            );
        } else {
            contents = message;
        }

        let className = 'save-button btn ' + btnClass;

        if (extraClasses) {
            className += ' ' + extraClasses;
        }

        return (
            <button
                id='saveSetting'
                className={className}
                disabled={disabled}
                {...props}
            >
                {contents}
            </button>
        );
    }
}

export default injectIntl(FormButton);

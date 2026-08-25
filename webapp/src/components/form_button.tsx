// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

type Props = {
    executing?: boolean;
    disabled?: boolean;
    executingMessage?: React.ReactNode;
    defaultMessage?: React.ReactNode;
    btnClass?: string;
    extraClasses?: string;
    saving?: boolean;
    savingMessage?: string;
    type?: string;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

export default class FormButton extends PureComponent<Props> {
    static defaultProps = {
        disabled: false,
        savingMessage: 'Creating',
        defaultMessage: 'Create',
        btnClass: 'btn-primary',
        extraClasses: '',
    };

    render() {
        const {saving, disabled, savingMessage, defaultMessage, btnClass, extraClasses, ...props} = this.props;

        let contents;
        if (saving) {
            contents = (
                <span>
                    <span
                        className='fa fa-spin fa-spinner'
                        title={'Loading Icon'}
                    />
                    {savingMessage}
                </span>
            );
        } else {
            contents = defaultMessage;
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

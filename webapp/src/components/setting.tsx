// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type Props = {
    inputId?: string;
    label?: React.ReactNode;
    children: React.ReactNode;
    helpText?: React.ReactNode;
    required?: boolean;
    hideRequiredStar?: boolean;
};

export default class Setting extends React.PureComponent<Props> {
    render() {
        const {
            children,
            helpText,
            inputId,
            label,
            required,
            hideRequiredStar,
        } = this.props;

        return (
            <div className='form-group less'>
                {label &&
                <label
                    className='control-label margin-bottom x2'
                    htmlFor={inputId}
                >
                    {label}
                </label>
                }
                {required && !hideRequiredStar &&
                <span
                    className='error-text'
                    style={{marginLeft: '3px'}}
                >
                    {'*'}
                </span>
                }
                <div>
                    {children}
                    <div className='help-text'>
                        {helpText}
                    </div>
                </div>
            </div>
        );
    }
}

// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

export default class Settings extends PureComponent {
    static propTypes = {
        inputId: PropTypes.string,
        label: PropTypes.node.isRequired,
        children: PropTypes.node.isRequired,
        helpText: PropTypes.node,
        required: PropTypes.bool,
    };

    render() {
        const {
            children,
            helpText,
            inputId,
            label,
            required,
        } = this.props;

        return (
            <div className='form-group'>
                <label
                    className='control-label'
                    htmlFor={inputId}
                >
                    {label}
                </label>
                {required &&
                <span style={{color: '#ff0000'}}>{'*'}</span>
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
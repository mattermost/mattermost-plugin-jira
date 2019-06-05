// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import ReactSelect from 'react-select';

import Setting from 'components/setting';

import {getStyleForReactSelect} from './utils';

export default class ReactSelectSetting extends React.PureComponent {
    static propTypes = {
        name: PropTypes.string.isRequired,
        onChange: PropTypes.func,
        theme: PropTypes.object.isRequired,
    };

    handleChange = (value) => {
        if (this.props.onChange) {
            if (Array.isArray(value)) {
                this.props.onChange(this.props.name, value.map((x) => x.value));
            } else {
                this.props.onChange(this.props.name, value.value);
            }
        }
    }

    render() {
        return (
            <Setting
                inputId={this.props.name}
                {...this.props}
            >
                <ReactSelect
                    {...this.props}
                    menuPortalTarget={document.body}
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            </Setting>
        );
    }
}

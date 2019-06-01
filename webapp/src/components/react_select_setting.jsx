// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import ReactSelect from 'react-select';

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import Setting from 'components/setting';

export default class ReactSelectSetting extends React.PureComponent {
    static propTypes = {
        name: PropTypes.string.isRequired,
        onChange: PropTypes.func,
        theme: PropTypes.object,
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

    getStyle = (theme) => ({
        menuPortal: (provided) => ({
            ...provided,
            zIndex: 9999,
        }),
        control: (provided, state) => ({
            ...provided,
            color: theme.centerChannelColor,
            background: theme.centerChannelBg,

            // Overwrittes the different states of border
            borderColor: state.isFocused ? changeOpacity(theme.centerChannelColor, 0.25) : changeOpacity(theme.centerChannelColor, 0.12),

            // Removes weird border around container
            boxShadow: 'inset 0 1px 1px ' + changeOpacity(theme.centerChannelColor, 0.075),
            borderRadius: '2px',

            '&:hover': {
                borderColor: changeOpacity(theme.centerChannelColor, 0.25),
            },
        }),
        option: (provided, state) => ({
            ...provided,
            background: state.isSelected ? changeOpacity(theme.centerChannelColor, 0.12) : theme.centerChannelBg,
            color: theme.centerChannelColor,
            '&:hover': {
                background: changeOpacity(theme.centerChannelColor, 0.12),
            },
        }),
        menu: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
            background: theme.centerChannelBg,
            border: '1px solid ' + changeOpacity(theme.centerChannelColor, 0.2),
            borderRadius: '0 0 2px 2px',
            boxShadow: changeOpacity(theme.centerChannelColor, 0.2) + ' 1px 3px 12px',
            marginTop: '4px',
        }),
        placeholder: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
        }),
        dropdownIndicator: (provided) => ({
            ...provided,
            color: changeOpacity(theme.centerChannelColor, 0.4),
        }),
        singleValue: (provided) => ({
            ...provided,
            color: theme.centerChannelColor,
        }),
        indicatorSeparator: (provided) => ({
            ...provided,
            display: 'none',
        }),
    });

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
                    styles={this.getStyle(this.props.theme)}
                />
            </Setting>
        );
    }
}

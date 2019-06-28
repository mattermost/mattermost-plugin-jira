// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

export const getStyleForReactSelect = (theme) => {
    if (!theme) {
        return null;
    }

    return {
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
            background: state.isFocused ? changeOpacity(theme.centerChannelColor, 0.12) : theme.centerChannelBg,
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
    };
};

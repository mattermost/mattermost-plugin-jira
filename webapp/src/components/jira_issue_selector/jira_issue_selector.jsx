// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {Component} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/lib/Async';

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

const searchDefaults = 'ORDER BY updated DESC';
const searchDebounceDelay = 400;

export default class JiraIssueSelector extends Component {
    static propTypes = {
        isRequired: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        currentProject: PropTypes.string.isRequired,
        onChange: PropTypes.func.isRequired,
        fetchIssuesEndpoint: PropTypes.string.isRequired,
    };

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = (text) => {
        const projectSearchTerm = this.props.currentProject ? 'project=' + this.props.currentProject : '';
        const textEncoded = encodeURIComponent(text.trim().replace(/"/g, '\\"'));
        const textSearchTerm = (textEncoded.length > 0) ? 'text ~ "' + textEncoded + '*"' : '';
        const combinedTerms = (projectSearchTerm.length > 0 && textSearchTerm.length > 0) ? projectSearchTerm + ' AND ' + textSearchTerm : projectSearchTerm + textSearchTerm;
        const finalQuery = combinedTerms + ' ' + searchDefaults;

        return fetch(this.props.fetchIssuesEndpoint + `?jql=${finalQuery}`).then(
            (response) => response.json()).then(
            (json) => {
                return json;
            });
    };

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

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

    render = () => {
        const requiredStar = (
            <span
                className={'error-text'}
                style={{marginLeft: '3px'}}
            >
                {'*'}
            </span>
        );

        return (
            <div className={'form-group margin-bottom x3'}>
                <label
                    className={'control-label'}
                    htmlFor={'issue'}
                >
                    {'Jira Issue'}
                </label>
                {this.props.isRequired && requiredStar}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={(e) => this.props.onChange(e ? e.value : '')}
                    required={true}
                    disabled={false}
                    isMulti={false}
                    isClearable={true}
                    defaultOptions={true}
                    loadOptions={this.handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    styles={this.getStyle(this.props.theme)}
                />
                <div className={'help-text'}>
                    {'Returns issues sorted by most recently updated.'} <br/>
                    {'Tip: Use AND, OR, *, ~, and other modifiers like in a JQL query.'}
                </div>
            </div>
        );
    }
}

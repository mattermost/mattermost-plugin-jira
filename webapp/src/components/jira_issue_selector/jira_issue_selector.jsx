// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {Component} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/lib/Async';

import {getStyleForReactSelect} from 'utils/styles';
import {doFetchWithResponse} from 'client';

const searchDefaults = 'ORDER BY updated DESC';
const searchDebounceDelay = 400;

export default class JiraIssueSelector extends Component {
    static propTypes = {
        isRequired: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        fetchIssuesEndpoint: PropTypes.string.isRequired,
    };

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = (text) => {
        const textEncoded = encodeURIComponent(text.trim().replace(/"/g, '\\"'));
        const textSearchTerm = (textEncoded.length > 0) ? 'text ~ "' + textEncoded + '*"' : '';
        const finalQuery = textSearchTerm + ' ' + searchDefaults;

        return doFetchWithResponse(this.props.fetchIssuesEndpoint + `?jql=${finalQuery}`).then(
            ({data}) => {
                return data;
            });
    };

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
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                <div className={'help-text'}>
                    {'Returns issues sorted by most recently updated.'} <br/>
                    {'Tip: Use AND, OR, *, ~, and other modifiers like in a JQL query.'}
                </div>
            </div>
        );
    }
}

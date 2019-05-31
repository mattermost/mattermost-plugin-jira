// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {Component} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/lib/Async';

const searchDefaults = 'ORDER BY updated DESC';
const searchDebounceDelay = 400;

export default class JiraIssueSelector extends Component {
    static propTypes = {
        isRequired: PropTypes.bool,
        currentProject: PropTypes.string.isRequired,
        onChange: PropTypes.func.isRequired,
        fetchIssuesEndpoint: PropTypes.string.isRequired,
    };

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = (text) => {
        const projectSearchTerm = this.props.currentProject ? 'project=' + this.props.currentProject : '';
        const textEncoded = encodeURIComponent(text.replace(/"/g, '\\"'));
        const textSearchTerm = (textEncoded.length > 0) ? 'text ~ "' + textEncoded + '*"' : '';
        const combinedTerms = (projectSearchTerm.length > 0 && textSearchTerm.length > 0) ? projectSearchTerm + ' AND ' + textSearchTerm : projectSearchTerm + textSearchTerm;
        const finalQuery = combinedTerms + ' ' + searchDefaults;

        return fetch(this.props.fetchIssuesEndpoint + `?jql=${finalQuery}`).then(
            (response) => response.json()).then(
            (json) => {
                return json;
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
            <div className={'form-group'}>
                <label
                    className={'control-label'}
                    htmlFor={'issue'}
                    aria-required={true}
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
                />
                <div className={'help-text'}>
                    {'Returns issues sorted by most recently updated.'}
                </div>
                <div className={'help-text'}>
                    {'Tip: Use AND, OR, *, ~, and other modifiers like in a JQL query.'}
                </div>
            </div>
        );
    }
}

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {Component} from 'react';
import type {Theme} from 'mattermost-redux/selectors/entities/preferences';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {getStyleForReactSelect} from 'utils/styles';

const searchDebounceDelay = 400;

type Props = {
    required?: boolean;
    theme: Theme;
    onChange: (value: string) => void;
    searchIssues: (params: Record<string, string>) => Promise<{data: any[]}>;
    error?: string;
    value?: string;
    addValidate: (fn: () => boolean) => void;
    removeValidate: (fn: () => boolean) => void;
    instanceID: string;
};

type State = {
    invalid: boolean;
    error?: string;
};

export default class JiraIssueSelector extends Component<Props, State> {
    constructor(props: Props) {
        super(props);
        this.state = {invalid: false};
    }

    componentDidMount() {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props, prevState: State) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleIssueSearchTermChange = (inputValue: string) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = (text: string) => {
        const params = {
            fields: 'key,summary',
            q: text.trim(),
            instance_id: this.props.instanceID,
        };

        return this.props.searchIssues(params).then(({data}) => {
            if (!data) {
                return [];
            }
            return data.map((issue) => ({
                value: issue.key,
                label: `${issue.key}: ${issue.fields.summary}`,
            }));
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

    onChange = (e: {value: string} | null) => {
        const value = e ? e.value : '';
        this.props.onChange(value);
    };

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value && this.props.value.toString().length !== 0);
        this.setState({invalid: !valid});
        return valid;
    };

    render = () => {
        const {error} = this.props;
        const requiredStar = (
            <span
                className={'error-text'}
                style={{marginLeft: '3px'}}
            >
                {'*'}
            </span>
        );

        let issueError = null;
        if (error) {
            issueError = (
                <p className='help-text error-text'>
                    <span>{error}</span>
                </p>
            );
        }

        const serverError = this.state.error;
        let errComponent;
        if (this.state.error) {
            errComponent = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {serverError?.toString()}</span>
                </p>
            );
        }

        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        return (
            <div className={'form-group less'}>
                {errComponent}
                <label
                    className={'control-label'}
                    htmlFor={'issue'}
                >
                    {'Jira Issue'}
                </label>
                {this.props.required && requiredStar}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={this.onChange}
                    required={true}
                    isDisabled={false}
                    isMulti={false}
                    isClearable={true}
                    defaultOptions={true}
                    loadOptions={this.handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {validationError}
                {issueError}
            </div>
        );
    };
}

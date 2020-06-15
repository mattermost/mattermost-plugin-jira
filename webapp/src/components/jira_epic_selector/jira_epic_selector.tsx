// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {getStyleForReactSelect} from 'utils/styles';
import {isEpicNameField, isEpicIssueType} from 'utils/jira_issue_metadata';
import {IssueMetadata, ReactSelectOption, JiraIssue, SearchIssueParams} from 'types/model';

import Setting from 'components/setting';

const searchDefaults = 'ORDER BY updated DESC';
const searchDebounceDelay = 400;

type Props = {
    required?: boolean;
    hideRequiredStar?: boolean;
    searchIssues: (params: SearchIssueParams) => Promise<{data: JiraIssue[]}>;
    theme: object;
    isMulti?: boolean;
    onChange: (values: string[]) => void;
    value: string[];
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    issueMetadata: IssueMetadata;
    resetInvalidOnChange?: boolean;
    instanceID: string;
};

type State = {
    invalid: boolean;
    error?: string;
    cachedSelectedOptions: ReactSelectOption[];
};

export default class JiraEpicSelector extends React.PureComponent<Props, State> {
    static defaultProps = {
        isMulti: false,
    };

    constructor(props: Props) {
        super(props);

        this.state = {
            cachedSelectedOptions: [],
            invalid: false,
        };
    }

    componentDidMount(): void {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
        this.fetchInitialSelectedValues();
    }

    componentWillUnmount(): void {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props, prevState: State): void {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    fetchInitialSelectedValues = (): void => {
        if (!this.props.value.length) {
            return;
        }

        const epicIds = this.props.value.join(', ');
        const searchStr = `and id IN (${epicIds})`;
        const userInput = ''; // Fetching by saved ids, no user input to process

        this.fetchEpicsFromJql(searchStr, userInput).then((options) => {
            if (options) {
                this.setState({cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options)});
            }
        });
    };

    handleIssueSearchTermChange = (inputValue: string): Promise<ReactSelectOption[]> => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = async (userInput: string): Promise<ReactSelectOption[]> => {
        const epicIssueType = this.props.issueMetadata.projects[0].issuetypes.find(isEpicIssueType);
        if (!epicIssueType) {
            return [];
        }

        const epicNameTypeId = Object.keys(epicIssueType.fields).find((key) => isEpicNameField(epicIssueType.fields[key]));
        if (!epicNameTypeId) {
            return [];
        }

        const epicNameTypeName = epicIssueType.fields[epicNameTypeId].name;

        let searchStr = '';
        if (userInput) {
            const cleanedInput = userInput.trim().replace(/"/g, '\\"');
            searchStr = ` and ("${epicNameTypeName}"~"${cleanedInput}" or "${epicNameTypeName}"~"${cleanedInput}*")`;
        }

        return this.fetchEpicsFromJql(searchStr, userInput);
    };

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

    fetchEpicsFromJql = async (jqlSearch: string, userInput: string): Promise<ReactSelectOption[]> => {
        const epicIssueType = this.props.issueMetadata.projects[0].issuetypes.find(isEpicIssueType);
        if (!epicIssueType) {
            return [];
        }

        const epicNameTypeId = Object.keys(epicIssueType.fields).find((key) => isEpicNameField(epicIssueType.fields[key]));
        if (!epicNameTypeId) {
            return [];
        }

        const projectKey = this.props.issueMetadata.projects[0].key;
        const fullJql = `project=${projectKey} and issuetype=${epicIssueType.id} ${jqlSearch} ${searchDefaults}`;

        const params = {
            jql: fullJql,
            fields: epicNameTypeId,
            q: userInput,
            instance_id: this.props.instanceID,
        };

        return this.props.searchIssues(params).then(({data}: {data: JiraIssue[]}) => {
            return data.map((issue) => ({
                value: issue.key,
                label: `${issue.key}: ${issue.fields[epicNameTypeId]}`,
            }));
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    onChange = (options: ReactSelectOption[]): void => {
        if (!options) {
            this.props.onChange([]);
            return;
        }
        this.setState({cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options)});
        this.props.onChange(options.map((v) => v.value));

        if (this.props.resetInvalidOnChange) {
            this.setState({invalid: false});
        }
    }

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }

        const valid = this.props.value && this.props.value.toString().length !== 0;
        this.setState({invalid: !valid});
        return valid;
    };

    render = (): JSX.Element => {
        const serverError = this.state.error;
        let errComponent;
        if (serverError) {
            errComponent = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {serverError.toString()}</span>
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

        const values = this.props.value.map((v) => {
            if (this.state.cachedSelectedOptions && this.state.cachedSelectedOptions.length) {
                const selected = this.state.cachedSelectedOptions.find((option) => option.value === v);
                if (selected) {
                    return selected;
                }
            }

            // Epic's name hasn't been fetched yet
            return {
                label: v,
                value: v,
            };
        });

        const {
            issueMetadata, // eslint-disable-line @typescript-eslint/no-unused-vars
            ...props
        } = this.props;

        const selectComponent = (
            <AsyncSelect
                {...props}
                name={'epic'}
                value={values}
                onChange={this.onChange}
                required={this.props.required}
                isMulti={this.props.isMulti}
                defaultOptions={true}
                loadOptions={this.handleIssueSearchTermChange}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                styles={getStyleForReactSelect(this.props.theme)}
            />
        );

        return (
            <Setting
                {...props}
                inputId={'epic'}
            >
                {selectComponent}
                {errComponent}
                {validationError}
            </Setting>
        );
    }
}

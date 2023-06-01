import React from 'react';

import {Theme} from 'mattermost-redux/types/preferences';

import ReactSelectSetting from 'components/react_select_setting';
import JiraEpicSelector from 'components/data_selectors/jira_epic_selector';

import {isEpicLinkField, isMultiSelectField, isLabelField, isSecurityLevelField} from 'utils/jira_issue_metadata';
import {FilterField, FilterValue, ReactSelectOption, IssueMetadata, IssueType, FilterFieldInclusion} from 'types/model';
import ConfirmModal from 'components/confirm_modal';
import JiraAutoCompleteSelector from 'components/data_selectors/jira_autocomplete_selector';

export type Props = {
    fields: FilterField[];
    field: FilterField;
    value: FilterValue;
    theme: Theme;
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    onChange: (f1: FilterValue, f2: FilterValue) => void;
    removeFilter: (f1: FilterValue) => void;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    instanceID: string;
    securityLevelEmptyForJiraSubscriptions: boolean;
};

export type State = {
    showConfirmDeleteModal: boolean;
    error: string | null;
}

export default class ChannelSubscriptionFilter extends React.PureComponent<Props, State> {
    state = {
        showConfirmDeleteModal: false,
        error: null,
    };

    componentDidMount() {
        this.props.addValidate(this.isValid);
    }

    componentWillUnmount() {
        this.props.removeValidate(this.isValid);
    }

    handleInclusionChange = (name: string, choice: FilterFieldInclusion): void => {
        const {onChange, value} = this.props;

        const newValues = choice === FilterFieldInclusion.EMPTY ? [] : value.values;

        onChange(value, {...value, inclusion: choice, values: newValues});
    };

    handleFieldTypeChange = (name: string, choice: string): void => {
        const {onChange, value} = this.props;

        onChange(value, {...value, values: [], key: choice, inclusion: FilterFieldInclusion.INCLUDE_ANY});
    };

    handleFieldValuesChange = (name: string, values: string[]): void => {
        const {onChange, value} = this.props;

        const newValues = values || [];
        onChange(value, {...value, values: newValues});
    };

    handleEpicLinkChange = (values: string[]): void => {
        const {onChange, value} = this.props;

        const newValues = values || [];
        onChange(value, {...value, values: newValues});
    };

    openDeleteModal = (): void => {
        this.setState({showConfirmDeleteModal: true});
    };

    handleCancelDelete = (): void => {
        this.setState({showConfirmDeleteModal: false});
    };

    handleConfirmDelete = (): void => {
        this.setState({showConfirmDeleteModal: false});
        this.removeFilter();
    };

    removeFilter = (): void => {
        this.props.removeFilter(this.props.value);
    };

    getConflictingIssueTypes = (): IssueType[] => {
        const conflictingIssueTypes = [];
        for (const issueTypeId of this.props.chosenIssueTypes) {
            if (this.props.field) {
                const issueTypes = this.props.field.issueTypes;
                if (!issueTypes.find((it) => it.id === issueTypeId)) {
                    const issueType = this.props.issueMetadata.projects[0].issuetypes.find((i) => i.id === issueTypeId) as IssueType;
                    conflictingIssueTypes.push(issueType);
                }
            }
        }
        return conflictingIssueTypes;
    };

    isValid = (): boolean => {
        let error = this.checkFieldConflictError();
        if (error) {
            this.setState({error});
            return false;
        }

        error = this.checkInclusionError();
        if (error) {
            this.setState({error});
            return false;
        }

        return true;
    }

    checkInclusionError = (): string | null => {
        const inclusion = this.props.value && this.props.value.inclusion;

        if (isSecurityLevelField(this.props.field) && inclusion === FilterFieldInclusion.EXCLUDE_ANY && this.props.securityLevelEmptyForJiraSubscriptions) {
            return 'Security level inclusion cannot be "Exclude Any". Note that the default value is now "Empty".';
        }

        return null;
    }

    checkFieldConflictError = (): string | null => {
        const conflictIssueTypes = this.getConflictingIssueTypes().map((it) => it.name);
        if (conflictIssueTypes.length) {
            return `${this.props.field.name} does not exist for issue type(s): ${conflictIssueTypes.join(', ')}.`;
        }
        return null;
    };

    renderInclusionDropdownOption = (data: ReactSelectOption, meta: {context: string}): JSX.Element | string => {
        const {value, label} = data;
        const {context} = meta;

        // context === value means it is rendering the selected value
        if (context === 'value') {
            return label;
        }

        // otherwise it is rendering an option in the open dropdown
        let subtext = '';
        switch (value) {
        case FilterFieldInclusion.INCLUDE_ANY:
            subtext = 'Includes either of the values (or)';
            break;
        case FilterFieldInclusion.INCLUDE_ALL:
            subtext = 'Includes all of the values (and)';
            break;
        case FilterFieldInclusion.EXCLUDE_ANY:
            subtext = 'Excludes all of the values';
            break;
        case FilterFieldInclusion.EMPTY:
            subtext = 'Includes when the value is empty';
            break;
        }

        return (
            <div>
                <div>{label}</div>
                <div style={{opacity: 0.6}}>
                    {subtext}
                </div>
            </div>
        );
    }

    render(): JSX.Element {
        const {field, fields, value, theme} = this.props;
        let chosenFieldValues: ReactSelectOption[] = [];
        const style = getStyle(theme);

        const fieldTypeOptions = fields.map((f) => ({
            value: f.key,
            label: f.name,
        }));
        let chosenFieldType = null;

        let inclusionSelectOptions: ReactSelectOption[] = [
            {label: 'Include', value: FilterFieldInclusion.INCLUDE_ANY},
            {label: 'Include All', value: FilterFieldInclusion.INCLUDE_ALL},
            {label: 'Exclude', value: FilterFieldInclusion.EXCLUDE_ANY},
            {label: 'Empty', value: FilterFieldInclusion.EMPTY},
        ];

        if (isSecurityLevelField(field) && value.inclusion !== FilterFieldInclusion.EXCLUDE_ANY && this.props.securityLevelEmptyForJiraSubscriptions) {
            inclusionSelectOptions = [
                {label: 'Include', value: FilterFieldInclusion.INCLUDE_ANY},
                {label: 'Include All', value: FilterFieldInclusion.INCLUDE_ALL},
                {label: 'Empty', value: FilterFieldInclusion.EMPTY},
            ];
        }

        if (!isMultiSelectField(field)) {
            const includeAllIndex = inclusionSelectOptions.findIndex((opt) => opt.value === FilterFieldInclusion.INCLUDE_ALL);
            inclusionSelectOptions.splice(includeAllIndex, 1);
        }

        let chosenInclusionOption = inclusionSelectOptions[0];

        const fieldValueOptions = (field && field.values) || [];

        if (field && value) {
            chosenInclusionOption = inclusionSelectOptions.find((opt) => opt.value === value.inclusion) as ReactSelectOption;
            chosenFieldType = fieldTypeOptions.find((option) => option.value === value.key);
            if (field.userDefined) {
                chosenFieldValues = value.values.map((option) => ({
                    label: option,
                    value: option,
                }));
            } else {
                chosenFieldValues = fieldValueOptions.filter((option: ReactSelectOption) =>
                    value.values.includes(option.value),
                );
            }
        }

        let deleteButton;
        if (field) {
            deleteButton = (
                <button
                    onClick={this.openDeleteModal}
                    className='style--none'
                    style={style.trashIcon}
                    type='button'
                >
                    <i className='fa fa-trash'/>
                </button>
            );
        } else {
            deleteButton = (
                <button
                    onClick={this.openDeleteModal}
                    className='btn btn-info'
                    type='button'
                >
                    {'Cancel'}
                </button>
            );
        }

        let disableLastSelect = false;
        let lastSelectPlaceholder;
        if (value.inclusion === FilterFieldInclusion.EMPTY) {
            lastSelectPlaceholder = '';
            disableLastSelect = true;
        }

        const selectProps = {
            instanceID: this.props.instanceID,
            required: !disableLastSelect,
            isDisabled: disableLastSelect,
            isClearable: false,
            placeholder: lastSelectPlaceholder,
            theme,
            resetInvalidOnChange: true,
            hideRequiredStar: true,
            isMulti: true,
            addValidate: this.props.addValidate,
            removeValidate: this.props.removeValidate,
        };

        let valueSelector;
        if (isEpicLinkField(this.props.field)) {
            valueSelector = (
                <JiraEpicSelector
                    {...selectProps}
                    issueMetadata={this.props.issueMetadata}
                    value={value.values}
                    onChange={this.handleEpicLinkChange}
                />
            );
        } else if (isLabelField(field)) {
            valueSelector = (
                <JiraAutoCompleteSelector
                    {...selectProps}
                    fieldName={field.name}
                    value={value.values}
                    onChange={this.handleEpicLinkChange}
                />
            );
        } else {
            valueSelector = (
                <ReactSelectSetting
                    {...selectProps}
                    name={'values'}
                    options={fieldValueOptions}
                    onChange={this.handleFieldValuesChange}
                    value={chosenFieldValues}
                    allowUserDefinedValue={Boolean(field && field.userDefined)}
                />
            );
        }

        const confirmDeleteModal = (
            <ConfirmModal
                cancelButtonText={'Cancel'}
                confirmButtonText={'Delete'}
                confirmButtonClass={'btn btn-danger'}
                hideCancel={false}
                message={'Are you sure you want to delete this filter?'}
                onCancel={this.handleCancelDelete}
                onConfirm={this.handleConfirmDelete}
                show={this.state.showConfirmDeleteModal}
                title={'Field Filter'}
            />
        );

        return (
            <div className='row'>
                <div className='col-md-12 col-sm-12'>
                    <div
                        className='help-text error-text'
                        style={style.conflictingError}
                    >
                        {this.state.error}
                    </div>
                </div>
                <div className='col-md-11 col-sm-12'>
                    <div className='row'>
                        <div className='col-md-4 col-sm-12'>
                            <ReactSelectSetting
                                name={'fieldtype'}
                                required={true}
                                hideRequiredStar={true}
                                options={fieldTypeOptions}
                                value={chosenFieldType}
                                onChange={this.handleFieldTypeChange}
                                theme={theme}
                                addValidate={this.props.addValidate}
                                removeValidate={this.props.removeValidate}
                            />
                        </div>
                        <div className='col-md-4 col-sm-12'>
                            <ReactSelectSetting
                                name={'inclusion'}
                                required={true}
                                hideRequiredStar={true}
                                options={inclusionSelectOptions}
                                onChange={this.handleInclusionChange}
                                value={chosenInclusionOption}
                                theme={theme}
                                addValidate={this.props.addValidate}
                                removeValidate={this.props.removeValidate}
                                formatOptionLabel={this.renderInclusionDropdownOption}
                            />
                        </div>
                        <div className='col-md-4 col-sm-12'>
                            {valueSelector}
                        </div>
                    </div>
                </div>
                <div className='col-md-1 col-sm-12 text-center'>
                    {deleteButton}
                </div>
                {confirmDeleteModal}
            </div>
        );
    }
}

type EmptyChannelSubscriptionFilterProps = {
    fields: FilterField[];
    theme: object;
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    onChange: (f1: FilterValue | null, f2: FilterValue) => void;
    cancelAdd: () => void;
};

export function EmptyChannelSubscriptionFilter(props: EmptyChannelSubscriptionFilterProps): JSX.Element {
    const handleFieldTypeChange = (name: string, choice: string): void => {
        const {onChange} = props;

        onChange(null, {values: [], key: choice, inclusion: FilterFieldInclusion.INCLUDE_ANY});
    };

    const {fields, theme} = props;
    const style = getStyle(theme);

    const fieldTypeOptions = fields.map((f) => ({
        value: f.key,
        label: f.name,
    }));

    return (
        <div className='row'>
            <div className='col-md-11 col-sm-12'>
                <div className='row'>
                    <div className='col-md-4 col-sm-12'>
                        <ReactSelectSetting
                            name={'fieldtype'}
                            options={fieldTypeOptions}
                            onChange={handleFieldTypeChange}
                            theme={theme}
                        />
                    </div>
                    <div className='col-md-4 col-sm-12'>
                        <ReactSelectSetting
                            name={'exclude'}
                            options={[]}
                            isDisabled={true}
                            theme={theme}
                        />
                    </div>
                    <div className='col-md-4 col-sm-12'>
                        <ReactSelectSetting
                            name={'values'}
                            options={[]}
                            isDisabled={true}
                            theme={theme}
                        />
                    </div>
                </div>
            </div>
            <div className='col-md-1 col-sm-12 text-center'>
                <button
                    onClick={props.cancelAdd}
                    className='style--none'
                    style={style.trashIcon}
                    type='button'
                >
                    <i className='fa fa-trash'/>
                </button>
            </div>
        </div>
    );
}

const getStyle = (theme: any): any => ({
    trashIcon: {
        color: theme.errorTextColor,
        fontSize: '20px',
        margin: '0.5rem 0 0',
    },
    conflictingError: {
        margin: '0 0 10px',
    },
});

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {TrashCanOutlineIcon} from '@mattermost/compass-icons/components';
import {Button} from '@mattermost/compass-ui/components/button';
import {ErrorMessage} from '@mattermost/compass-ui/components/error-message';
import {Icon} from '@mattermost/compass-ui/components/icon';
import {IconButton} from '@mattermost/compass-ui/components/icon-button';

import SelectSetting from 'components/select_setting';
import WithTooltip from 'components/with_tooltip';
import JiraEpicSelector from 'components/data_selectors/jira_epic_selector';
import JiraSprintSelector from 'components/data_selectors/jira_sprint_selector';

import {
    FIELD_KEY_STATUS,
    isCommentVisibilityField,
    isEpicLinkField,
    isLabelField,
    isMultiSelectField,
    isSecurityLevelField,
    isSprintField,
    isTeamField,
    isUserField,
} from 'utils/jira_issue_metadata';
import {
    FilterField,
    FilterFieldInclusion,
    FilterValue,
    IssueMetadata,
    IssueType,
    SelectOption,
} from 'types/model';
import ConfirmModal from 'components/confirm_modal';
import JiraAutoCompleteSelector from 'components/data_selectors/jira_autocomplete_selector';
import JiraCommentVisibilitySelector from 'components/data_selectors/jira_commentvisibility_selector';
import JiraTeamSelector from 'components/data_selectors/jira_team_selector';

export type Props = {
    fields: FilterField[];
    field: FilterField;
    value: FilterValue;
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    onChange: (f1: FilterValue, f2: FilterValue) => void;
    removeFilter: (f1: FilterValue) => void;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    instanceID: string;
    securityLevelEmptyForJiraSubscriptions?: boolean;
    searchTeamFields: (params: {fieldValue: string; instance_id: string}) => Promise<{data: {name: string; id: string}[]}>;
    projectKey: string;
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

    handleTeamSelection = (selected: string | string[]): void => {
        const {onChange, value} = this.props;

        const selectedValue = Array.isArray(selected) ? selected[0] : selected;

        if (!selectedValue) {
            onChange(value, {...value, values: []});
            return;
        }

        onChange(value, {...value, values: [selectedValue]});
    };

    handleValueChangeWithoutName = (values: string[]): void => {
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
    };

    checkInclusionError = (): string | null => {
        const inclusion = this.props.value && this.props.value.inclusion;

        if (isSecurityLevelField(this.props.field) && inclusion === FilterFieldInclusion.EXCLUDE_ANY && this.props.securityLevelEmptyForJiraSubscriptions) {
            return 'Security level inclusion cannot be "Exclude Any". Note that the default value is now "Empty".';
        }

        return null;
    };

    checkFieldConflictError = (): string | null => {
        const conflictIssueTypes = this.getConflictingIssueTypes().map((it) => it.name);
        if (conflictIssueTypes.length) {
            return `${this.props.field.name} does not exist for issue type(s): ${conflictIssueTypes.join(', ')}.`;
        }
        return null;
    };

    render(): JSX.Element {
        const {field, fields, value} = this.props;
        let chosenFieldValues: SelectOption[] = [];
        const fieldTypeOptions = fields.map((f) => ({
            value: f.key,
            label: f.name,
        }));
        let chosenFieldType = null;

        let inclusionSelectOptions: SelectOption[] = [
            {label: 'Include', value: FilterFieldInclusion.INCLUDE_ANY},
            {label: 'Include All', value: FilterFieldInclusion.INCLUDE_ALL},
            {label: 'Exclude', value: FilterFieldInclusion.EXCLUDE_ANY},
            {label: 'Empty', value: FilterFieldInclusion.EMPTY},
            {label: 'Include or Empty', value: FilterFieldInclusion.INCLUDE_OR_EMPTY},
        ];

        if (isSecurityLevelField(field) && value.inclusion !== FilterFieldInclusion.EXCLUDE_ANY && this.props.securityLevelEmptyForJiraSubscriptions) {
            inclusionSelectOptions = [
                {label: 'Include', value: FilterFieldInclusion.INCLUDE_ANY},
                {label: 'Include All', value: FilterFieldInclusion.INCLUDE_ALL},
                {label: 'Empty', value: FilterFieldInclusion.EMPTY},
            ];
        }

        if (field.key === FIELD_KEY_STATUS) {
            inclusionSelectOptions = [
                {label: 'Include', value: FilterFieldInclusion.INCLUDE_ANY},
                {label: 'Exclude', value: FilterFieldInclusion.EXCLUDE_ANY},
            ];
        }

        if (!isMultiSelectField(field)) {
            const includeAllIndex = inclusionSelectOptions.findIndex((opt) => opt.value === FilterFieldInclusion.INCLUDE_ALL);
            inclusionSelectOptions.splice(includeAllIndex, 1);
        }

        inclusionSelectOptions = inclusionSelectOptions.map((opt) => ({...opt, secondaryLabel: INCLUSION_DESCRIPTIONS[opt.value]}));

        let chosenInclusionOption = inclusionSelectOptions[0];

        const fieldValueOptions = (field && field.values) || [];

        if (field && value) {
            chosenInclusionOption = inclusionSelectOptions.find((opt) => opt.value === value.inclusion) as SelectOption;
            chosenFieldType = fieldTypeOptions.find((option) => option.value === value.key);
            if (field.userDefined) {
                chosenFieldValues = value.values.map((option) => ({
                    label: option,
                    value: option,
                }));
            } else {
                chosenFieldValues = fieldValueOptions.filter((option: SelectOption) =>
                    value.values.includes(option.value),
                );
            }
        }

        let deleteButton;
        if (field) {
            deleteButton = (
                <WithTooltip label='Delete filter'>
                    <IconButton
                        aria-label='Delete filter'
                        destructive={true}
                        icon={<Icon glyph={<TrashCanOutlineIcon/>}/>}
                        onClick={this.openDeleteModal}
                    />
                </WithTooltip>
            );
        } else {
            deleteButton = (
                <Button
                    emphasis='tertiary'
                    size='small'
                    onClick={this.openDeleteModal}
                >
                    {'Cancel'}
                </Button>
            );
        }

        let disableLastSelect = false;
        let lastSelectPlaceholder = 'Values';
        if (value.inclusion === FilterFieldInclusion.EMPTY) {
            lastSelectPlaceholder = '';
            disableLastSelect = true;
        }

        const selectProps = {
            'aria-label': 'Values',
            instanceID: this.props.instanceID,
            required: !disableLastSelect,
            isDisabled: disableLastSelect,
            isClearable: false,
            placeholder: lastSelectPlaceholder,
            resetInvalidOnChange: true,
            hideRequiredStar: true,
            isMulti: true,
            addValidate: this.props.addValidate,
            removeValidate: this.props.removeValidate,
        };

        let valueSelector;
        if (isCommentVisibilityField(field)) {
            valueSelector = (
                <JiraCommentVisibilitySelector
                    {...selectProps}
                    fieldName={field.name}
                    value={value.values}
                    onChange={this.handleValueChangeWithoutName}
                />
            );
        } else if (isEpicLinkField(this.props.field)) {
            valueSelector = (
                <JiraEpicSelector
                    {...selectProps}
                    issueMetadata={this.props.issueMetadata}
                    value={value.values}
                    onChange={this.handleValueChangeWithoutName}
                />
            );
        } else if (isLabelField(field)) {
            valueSelector = (
                <JiraAutoCompleteSelector
                    {...selectProps}
                    fieldName={field.name}
                    value={value.values}
                    onChange={this.handleValueChangeWithoutName}
                />
            );
        } else if (isTeamField(field)) {
            valueSelector = (
                <JiraTeamSelector
                    {...selectProps}
                    fieldName={field.name}
                    value={value.values[0] || ''}
                    onChange={this.handleTeamSelection}
                    searchTeamFields={this.props.searchTeamFields}
                />
            );
        } else if (isSprintField(field)) {
            valueSelector = (
                <JiraSprintSelector
                    {...selectProps}
                    isMulti={false}
                    value={value.values[0] || ''}
                    onChange={(selected: string | string[]) => {
                        const selectedValue = Array.isArray(selected) ? selected : [selected];
                        this.handleValueChangeWithoutName(selectedValue.filter(Boolean));
                    }}
                    projectKey={this.props.projectKey}
                />
            );
        } else if (isUserField(field)) {
            valueSelector = (
                <JiraAutoCompleteSelector
                    {...selectProps}
                    fieldName={field.name}
                    value={value.values}
                    onChange={this.handleValueChangeWithoutName}
                />
            );
        } else {
            valueSelector = (
                <SelectSetting
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
                destructive={true}
                hideCancel={false}
                message={'Are you sure you want to delete this filter?'}
                onCancel={this.handleCancelDelete}
                onConfirm={this.handleConfirmDelete}
                show={this.state.showConfirmDeleteModal}
                title={'Field Filter'}
            />
        );

        return (
            <div className='jira-subscription-filter'>
                {this.state.error && (
                    <ErrorMessage
                        className='jira-subscription-filter__error'
                        message={this.state.error}
                    />
                )}
                <div className='jira-subscription-filter__row'>
                    <SelectSetting
                        name={'fieldtype'}
                        aria-label='Field'
                        placeholder='Field'
                        required={true}
                        hideRequiredStar={true}
                        options={fieldTypeOptions}
                        value={chosenFieldType}
                        onChange={this.handleFieldTypeChange}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                    />
                    <SelectSetting
                        name={'inclusion'}
                        aria-label='Inclusion'
                        placeholder='Inclusion'
                        required={true}
                        hideRequiredStar={true}
                        options={inclusionSelectOptions}
                        onChange={this.handleInclusionChange}
                        value={chosenInclusionOption}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                    />
                    {valueSelector}
                    <div className='jira-subscription-filter__action'>
                        {deleteButton}
                    </div>
                </div>
                {confirmDeleteModal}
            </div>
        );
    }
}

type EmptyChannelSubscriptionFilterProps = {
    fields: FilterField[];
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

    const fieldTypeOptions = props.fields.map((f) => ({
        value: f.key,
        label: f.name,
    }));

    return (
        <div className='jira-subscription-filter'>
            <div className='jira-subscription-filter__row'>
                <SelectSetting
                    name={'fieldtype'}
                    aria-label='Field'
                    placeholder='Field'
                    options={fieldTypeOptions}
                    onChange={handleFieldTypeChange}
                />
                <SelectSetting
                    name={'exclude'}
                    aria-label='Inclusion'
                    placeholder='Inclusion'
                    options={[]}
                    isDisabled={true}
                />
                <SelectSetting
                    name={'values'}
                    aria-label='Values'
                    placeholder='Values'
                    options={[]}
                    isDisabled={true}
                />
                <div className='jira-subscription-filter__action'>
                    <WithTooltip label='Remove filter'>
                        <IconButton
                            aria-label='Remove filter'
                            destructive={true}
                            icon={<Icon glyph={<TrashCanOutlineIcon/>}/>}
                            onClick={props.cancelAdd}
                        />
                    </WithTooltip>
                </div>
            </div>
        </div>
    );
}

const INCLUSION_DESCRIPTIONS: Record<string, string> = {
    [FilterFieldInclusion.INCLUDE_ANY]: 'Includes either of the values (or)',
    [FilterFieldInclusion.INCLUDE_ALL]: 'Includes all of the values (and)',
    [FilterFieldInclusion.EXCLUDE_ANY]: 'Excludes all of the values',
    [FilterFieldInclusion.EMPTY]: 'Includes when the value is empty',
    [FilterFieldInclusion.INCLUDE_OR_EMPTY]: 'Includes the specified values or when the value is empty',
};

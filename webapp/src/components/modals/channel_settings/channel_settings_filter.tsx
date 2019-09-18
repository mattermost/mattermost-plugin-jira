import React from 'react';

import ReactSelectSetting from 'components/react_select_setting';

import {FilterField, FilterValue, ReactSelectOption, IssueMetadata, IssueType} from 'types/model';

type ChannelSettingsFilterProps = {
    fields: FilterField[];
    field: FilterField;
    value: FilterValue;
    theme: object;
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    onChange: (f1: FilterValue, f2: FilterValue) => void;
    removeFilter: (f1: FilterValue) => void;
    addValidate: (name: string | null, isValid: () => boolean) => void;
    removeValidate: (name: string | null, isValid: () => boolean) => void;
};

export default class ChannelSettingsFilter extends React.PureComponent<ChannelSettingsFilterProps> {
    componentDidMount() {
        this.props.addValidate(null, this.isValid);
    }

    componentWillUnmount() {
        this.props.removeValidate(null, this.isValid);
    }

    handleExcludeChange = (name: string, choice: string): void => {
        const {onChange, value} = this.props;

        const newValue = choice === '1';
        onChange(value, {...value, exclude: newValue});
    };

    handleFieldTypeChange = (name: string, choice: string): void => {
        const {onChange, value} = this.props;

        onChange(value, {...value, values: [], key: choice, exclude: false});
    };

    handleFieldValueChange = (name: string, values: string[]): void => {
        const {onChange, value} = this.props;

        const newValues = values || [];
        onChange(value, {...value, values: newValues});
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

    isOptionDisabled = (option: ReactSelectOption) => {
        return false;
    }

    isValid = (): boolean => {
        const error = this.checkFieldConflictError();
        if (error) {
            this.setState({error});
            return false;
        }

        return true;
    }

    checkFieldConflictError = (): string | null => {
        const conflictIssueTypes = this.getConflictingIssueTypes().map((it) => it.name);
        if (conflictIssueTypes.length) {
            return `Error: ${this.props.field.name} does not exist for issue type(s): ${conflictIssueTypes.join(', ')}.`;
        }
        return null;
    };

    render(): JSX.Element {
        const {field, fields, value, theme} = this.props;
        let chosenFieldValues: ReactSelectOption[] = [];

        const fieldTypeOptions = fields.map((f) => ({
            value: f.key,
            label: f.name,
        }));
        let chosenFieldType = null;

        const excludeSelectOptions = [
            {label: 'Include', value: '0'},
            {label: 'Exclude', value: '1'},
        ];
        let chosenExcludeValue = excludeSelectOptions[0];

        const fieldValueOptions = (field && field.values) || [];

        if (field && value) {
            chosenExcludeValue = value.exclude ? excludeSelectOptions[1] : excludeSelectOptions[0];
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
                    onClick={this.removeFilter}
                    className='btn btn-danger'
                >
                    {'Remove'}
                </button>
            );
        } else {
            deleteButton = (
                <button
                    onClick={this.removeFilter}
                    className='btn btn-info'
                >
                    {'Cancel'}
                </button>
            );
        }

        return (
            <div>
                <div>
                    <span>
                        {this.checkFieldConflictError()}
                    </span>
                </div>
                <div style={{width: '30%', display: 'inline-block'}}>
                    <ReactSelectSetting
                        name={'fieldtype'}
                        required={true}
                        hideRequiredStar={true}
                        options={fieldTypeOptions}
                        isOptionDisabled={this.isOptionDisabled}
                        value={chosenFieldType}
                        onChange={this.handleFieldTypeChange}
                        theme={theme}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                    />
                </div>
                <div style={{width: '30%', display: 'inline-block'}}>
                    <ReactSelectSetting
                        name={'exclude'}
                        required={true}
                        hideRequiredStar={true}
                        options={excludeSelectOptions}
                        onChange={this.handleExcludeChange}
                        value={chosenExcludeValue}
                        theme={theme}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                    />
                </div>
                <div style={{width: '30%', display: 'inline-block'}}>
                    <ReactSelectSetting
                        name={'values'}
                        required={true}
                        hideRequiredStar={true}
                        options={fieldValueOptions}
                        theme={theme}
                        onChange={this.handleFieldValueChange}
                        value={chosenFieldValues}
                        isMulti={true}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                        allowUserDefinedValue={Boolean(field && field.userDefined)}
                    />
                </div>
                {deleteButton}
            </div>
        );
    }
}

type EmptyChannelSettingsFilterProps = {
    fields: FilterField[];
    theme: object;
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    onChange: (f1: FilterValue | null, f2: FilterValue) => void;
    cancelAdd: () => void;
};

export function EmptyChannelSettingsFilter(props: EmptyChannelSettingsFilterProps) {
    const handleFieldTypeChange = (name: string, choice: string): void => {
        const {onChange} = props;

        onChange(null, {values: [], key: choice, exclude: false});
    };

    const {fields, theme} = props;

    const fieldTypeOptions = fields.map((f) => ({
        value: f.key,
        label: f.name,
    }));

    return (
        <div>
            <div style={{width: '30%', display: 'inline-block'}}>
                <ReactSelectSetting
                    name={'fieldtype'}
                    options={fieldTypeOptions}
                    onChange={handleFieldTypeChange}
                    theme={theme}
                />
            </div>
            <div style={{width: '30%', display: 'inline-block'}}>
                <ReactSelectSetting
                    name={'exclude'}
                    options={[]}
                    isDisabled={true}
                    theme={theme}
                />
            </div>
            <div style={{width: '30%', display: 'inline-block'}}>
                <ReactSelectSetting
                    name={'values'}
                    options={[]}
                    isDisabled={true}
                    theme={theme}
                />
            </div>
            <button
                onClick={props.cancelAdd}
                className='btn btn-info'
            >
                {'Cancel'}
            </button>
        </div>
    );
}

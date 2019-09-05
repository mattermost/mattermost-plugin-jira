import React from 'react';

import ReactSelectSetting from 'components/react_select_setting';

import {FilterField, FilterValue, ReactSelectOption} from 'types/model';

type ChannelSettingsFilterProps = {
    fields: FilterField[];
    field: FilterField | null;
    value: FilterValue | null;
    theme: object;
    onChange: (f1: FilterValue | null, f2: FilterValue) => void;
    removeFilter: (f1: FilterValue | null) => void;
    addValidate?: () => void;
    removeValidate?: () => void;
};

export default class ChannelSettingsFilter extends React.PureComponent<ChannelSettingsFilterProps> {
    handleExcludeChange = (name: string, choice: string) => {
        const {onChange, value} = this.props;
        if (!value) {
            return;
        }

        const newValue = choice === '1';
        onChange(value, {...value, exclude: newValue});
    };

    handleFieldTypeChange = (name: string, choice: string) => {
        const {onChange, value} = this.props;

        onChange(value, {...value, values: [], key: choice, exclude: false});
    };

    handleFieldValueChange = (name: string, values: string[]) => {
        const {onChange, value} = this.props;
        if (!value) {
            return;
        }

        const newValues = values || [];
        onChange(value, {...value, values: newValues});
    };

    removeFilter = () => {
        this.props.removeFilter(this.props.value);
    };

    render() {
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

        return (
            <div>
                <div style={{width: '30%', display: 'inline-block'}}>
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
                <div style={{width: '30%', display: 'inline-block'}}>
                    <ReactSelectSetting
                        name={'exclude'}
                        required={true}
                        hideRequiredStar={true}
                        options={excludeSelectOptions}
                        onChange={this.handleExcludeChange}
                        value={chosenExcludeValue}
                        disabled={!value}
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
                        disabled={!value}
                        isMulti={true}
                        addValidate={this.props.addValidate}
                        removeValidate={this.props.removeValidate}
                        allowUserDefinedValue={Boolean(field && field.userDefined)}
                    />
                </div>

                {field ? (
                    <button
                        onClick={this.removeFilter}
                        className='btn btn-danger'
                    >
                        {'Remove'}
                    </button>
                ) : (
                    <button
                        onClick={this.removeFilter}
                        className='btn btn-info'
                    >
                        {'Cancel'}
                    </button>
                )}
            </div>
        );
    }
}

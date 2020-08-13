import React from 'react';

import {FilterField, FilterValue, IssueMetadata, FilterFieldInclusion} from 'types/model';

import {getConflictingFields} from 'utils/jira_issue_metadata';

import ChannelSettingsFilter, {EmptyChannelSettingsFilter} from './channel_settings_filter';

export type Props = {
    fields: FilterField[];
    values: FilterValue[];
    theme: {};
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    onChange: (f: FilterValue[]) => void;
    instanceID: string;
};

type State = {
    showCreateRow: boolean;
};

export default class ChannelSettingsFilters extends React.PureComponent<Props, State> {
    state = {
        showCreateRow: false,
    };

    onConfiguredValueChange = (oldValue: FilterValue | null, newValue: FilterValue): void => {
        const newValues = this.props.values.concat([]);
        const index = newValues.findIndex((f) => f === oldValue);

        if (index === -1) {
            newValues.push({inclusion: FilterFieldInclusion.INCLUDE_ANY, values: [], ...newValue});
            this.setState({showCreateRow: false});
        } else {
            newValues.splice(index, 1, newValue);
        }

        this.props.onChange(newValues);
    };

    addNewFilter = (): void => {
        this.setState({showCreateRow: true});
    };

    hideNewFilter = (): void => {
        this.setState({showCreateRow: false});
    };

    removeFilter = (value: FilterValue | null): void => {
        if (!value) {
            return;
        }

        const newValues = this.props.values.concat([]);
        const index = newValues.findIndex((f) => f === value);

        if (index !== -1) {
            newValues.splice(index, 1);
        }
        this.props.onChange(newValues);
    };

    render(): JSX.Element {
        const {fields, values} = this.props;
        const {showCreateRow} = this.state;
        const style = getStyle();

        const conflictingFields = getConflictingFields(
            this.props.fields,
            this.props.chosenIssueTypes,
            this.props.issueMetadata
        );
        const nonConflictingFields = fields.filter((f) => {
            return !conflictingFields.find((conf) => conf.field.key === f.key);
        });

        return (
            <div className='margin-bottom'>
                <label
                    className='control-label margin-bottom'
                >
                    {'Filters'}
                </label>
                <div>
                    {values.map((v, i) => {
                        const field = fields.find((f) => f.key === v.key);
                        if (!field) {
                            return null;
                        }
                        return (
                            <div key={i}>
                                <ChannelSettingsFilter
                                    fields={nonConflictingFields}
                                    field={field}
                                    value={v}
                                    chosenIssueTypes={this.props.chosenIssueTypes}
                                    issueMetadata={this.props.issueMetadata}
                                    onChange={this.onConfiguredValueChange}
                                    removeFilter={this.removeFilter}
                                    theme={this.props.theme}
                                    addValidate={this.props.addValidate}
                                    removeValidate={this.props.removeValidate}
                                    instanceID={this.props.instanceID}
                                />
                            </div>
                        );
                    })}
                    {showCreateRow && (
                        <div>
                            <EmptyChannelSettingsFilter
                                fields={nonConflictingFields}
                                chosenIssueTypes={this.props.chosenIssueTypes}
                                issueMetadata={this.props.issueMetadata}
                                onChange={this.onConfiguredValueChange}
                                theme={this.props.theme}
                                cancelAdd={this.hideNewFilter}
                            />
                        </div>
                    )}
                    <button
                        onClick={this.addNewFilter}
                        disabled={showCreateRow}
                        className='btn style--none d-flex align-items-center'
                        type='button'
                    >
                        <span style={style.plusIcon}>{'+'}</span>
                        {'Add Filter'}
                    </button>
                </div>
            </div>
        );
    }
}

const getStyle = () => ({
    plusIcon: {
        fontSize: '24px',
        margin: '-1px 4px 0 0',
    },
});

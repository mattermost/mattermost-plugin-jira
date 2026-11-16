// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {PlusIcon} from '@mattermost/compass-icons/components';
import {Button} from '@mattermost/compass-ui/components/button';
import {Icon} from '@mattermost/compass-ui/components/icon';

import {
    FilterField,
    FilterFieldInclusion,
    FilterValue,
    IssueMetadata,
} from 'types/model';

import {getConflictingFields} from 'utils/jira_issue_metadata';

import ChannelSubscriptionFilter, {EmptyChannelSubscriptionFilter} from './channel_subscription_filter';

export type Props = {
    fields: FilterField[];
    values: FilterValue[];
    chosenIssueTypes: string[];
    issueMetadata: IssueMetadata;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    onChange: (f: FilterValue[]) => void;
    instanceID: string;
    securityLevelEmptyForJiraSubscriptions?: boolean;
    searchTeamFields: (params: {fieldValue: string; instance_id: string}) => Promise<{data: {name: string; id: string}[]}>;
    projectKey: string;
};

type State = {
    showCreateRow: boolean;
};

export default class ChannelSubscriptionFilters extends React.PureComponent<Props, State> {
    state = {
        showCreateRow: false,
    };

    onConfiguredValueChange = (oldValue: FilterValue | null, newValue: FilterValue): void => {
        const newValues = this.props.values.concat([]);
        const index = newValues.findIndex((f) => f === oldValue);

        if (index === -1) {
            newValues.push({...newValue, inclusion: FilterFieldInclusion.INCLUDE_ANY, values: []});
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

        const conflictingFields = getConflictingFields(
            this.props.fields,
            this.props.chosenIssueTypes,
            this.props.issueMetadata,
        );
        const nonConflictingFields = fields.filter((f) => {
            return !conflictingFields.find((conf) => conf.field.key === f.key);
        });

        return (
            <div className='jira-subscriptions__filters'>
                <h4 className='jira-subscriptions__filters-label'>
                    {'Filters'}
                </h4>
                <div>
                    {values.map((v, i) => {
                        const field = fields.find((f) => f.key === v.key);
                        if (!field) {
                            return null;
                        }
                        return (
                            <div key={i}>
                                <ChannelSubscriptionFilter
                                    fields={nonConflictingFields}
                                    field={field}
                                    value={v}
                                    chosenIssueTypes={this.props.chosenIssueTypes}
                                    issueMetadata={this.props.issueMetadata}
                                    onChange={this.onConfiguredValueChange}
                                    removeFilter={this.removeFilter}
                                    addValidate={this.props.addValidate}
                                    removeValidate={this.props.removeValidate}
                                    instanceID={this.props.instanceID}
                                    securityLevelEmptyForJiraSubscriptions={this.props.securityLevelEmptyForJiraSubscriptions}
                                    searchTeamFields={this.props.searchTeamFields}
                                    projectKey={this.props.projectKey}
                                />
                            </div>
                        );
                    })}
                    {showCreateRow && (
                        <div>
                            <EmptyChannelSubscriptionFilter
                                fields={nonConflictingFields}
                                chosenIssueTypes={this.props.chosenIssueTypes}
                                issueMetadata={this.props.issueMetadata}
                                onChange={this.onConfiguredValueChange}
                                cancelAdd={this.hideNewFilter}
                            />
                        </div>
                    )}
                    <Button
                        emphasis='tertiary'
                        size='small'
                        leadingIcon={<Icon glyph={<PlusIcon/>}/>}
                        onClick={this.addNewFilter}
                        disabled={showCreateRow}
                    >
                        {'Add Filter'}
                    </Button>
                </div>
            </div>
        );
    }
}

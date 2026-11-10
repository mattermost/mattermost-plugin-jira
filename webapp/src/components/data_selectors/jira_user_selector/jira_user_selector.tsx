// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {AvatarSize, JiraUser, SelectOption} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

type Props = Omit<BackendSelectorProps, 'value' | 'onChange' | 'search' | 'fetchInitialSelectedValues'> & {
    projectKey: string;
    value?: JiraUser | null;
    onChange: (user: JiraUser | null) => void;
    searchUsers: (params: {project: string; q: string; instance_id: string}) => Promise<{data: JiraUser[]; error?: Error}>;
};

const userKey = (user: JiraUser) => user.accountId || user.name || user.displayName;

const userToOption = (user: JiraUser): SelectOption => {
    const option: SelectOption = {value: userKey(user), label: user.displayName};
    const avatarURL = user.avatarUrls?.[AvatarSize.SMALL];
    if (avatarURL) {
        option.leadingAvatar = {src: avatarURL, alt: user.displayName};
    }
    return option;
};

export default class JiraUserSelector extends React.PureComponent<Props> {
    private usersByKey = new Map<string, JiraUser>();

    fetchInitialSelectedValues = async (): Promise<SelectOption[]> => {
        if (!this.props.value) {
            return [];
        }

        this.usersByKey.set(userKey(this.props.value), this.props.value);
        return [userToOption(this.props.value)];
    };

    searchUsers = (inputValue: string): Promise<SelectOption[]> => {
        const params = {
            q: inputValue,
            project: this.props.projectKey,
            instance_id: this.props.instanceID,
        };

        return this.props.searchUsers(params).then(({data, error}) => {
            if (error) {
                return [];
            }

            return data.map((user) => {
                this.usersByKey.set(userKey(user), user);
                return userToOption(user);
            });
        }).catch(() => {
            throw new Error('Error fetching data');
        });
    };

    handleChange = (key: string | string[]) => {
        const selected = Array.isArray(key) ? key[0] : key;
        this.props.onChange((selected && this.usersByKey.get(selected)) || null);
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...this.props}
                value={this.props.value ? userKey(this.props.value) : ''}
                onChange={this.handleChange}
                fetchInitialSelectedValues={this.fetchInitialSelectedValues}
                search={this.searchUsers}
            />
        );
    };
}

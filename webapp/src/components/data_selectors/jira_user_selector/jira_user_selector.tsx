// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ReactSelectOption, JiraUser, AvatarSize} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

type Props = BackendSelectorProps & {
    projectKey: string;
    searchUsers: (params: {project: string; q: string}) => Promise<{data: {label: string; value: JiraUser[]}; error?: Error}>;
};

export default class JiraUserSelector extends React.PureComponent<Props> {
    fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!this.props.value || (this.props.isMulti && !this.props.value.length)) {
            return [];
        }

        return this.searchUsers('');
    };

    searchUsers = (inputValue: string): Promise<ReactSelectOption[]> => {
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
                let label: string | React.ReactElement = user.displayName;
                const avatarURL = user.avatarUrls[AvatarSize.SMALL];
                if (avatarURL) {
                    label = (
                        <span>
                            <img
                                src={avatarURL}
                                style={{width: '24px', marginRight: '10px'}}
                            />
                            <span>{user.displayName}</span>
                        </span>
                    );
                }

                return {
                    value: user,
                    label,
                };
            });
        });
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...this.props}
                fetchInitialSelectedValues={this.fetchInitialSelectedValues}
                search={this.searchUsers}
            />
        );
    }
}

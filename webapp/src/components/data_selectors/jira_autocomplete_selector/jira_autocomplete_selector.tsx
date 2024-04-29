// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ReactSelectOption} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '/src/components/data_selectors/backend_selector';

const stripHTML = (text: string) => {
    if (!text) {
        return text;
    }

    const doc = new DOMParser().parseFromString(text, 'text/html');
    return doc.body.textContent || '';
};

type Props = BackendSelectorProps & {
    searchAutoCompleteFields: (params: {fieldValue: string; fieldName: string}) => (
        Promise<{data: {results: {value: string; displayName: string}[]}; error?: Error}>
    );
    fieldName: string;
};

export default class JiraAutoCompleteSelector extends React.PureComponent<Props> {
    fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!this.props.value || (this.props.isMulti && !this.props.value.length)) {
            return [];
        }

        return this.searchAutoCompleteFields('');
    };

    searchAutoCompleteFields = (inputValue: string): Promise<ReactSelectOption[]> => {
        const {fieldName, instanceID} = this.props;
        const params = {
            fieldValue: inputValue,
            fieldName,
            instance_id: instanceID,
        };
        return this.props.searchAutoCompleteFields(params).then(({data}) => {
            return data.results.map((suggestion) => ({
                value: suggestion.value,
                label: stripHTML(suggestion.displayName),
            }));
        }).catch((e) => {
            throw new Error('Error fetching data');
        });
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...this.props}
                fetchInitialSelectedValues={this.fetchInitialSelectedValues}
                search={this.searchAutoCompleteFields}
            />
        );
    };
}

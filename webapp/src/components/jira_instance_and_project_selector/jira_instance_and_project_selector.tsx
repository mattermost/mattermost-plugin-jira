import React from 'react';

import {Instance, ProjectMetadata, ReactSelectOption} from 'types/model';
import ReactSelectSetting from 'components/react_select_setting';
import {getProjectValues} from 'utils/jira_issue_metadata';

type Props = {
    selectedInstanceID: string | null;
    selectedProjectID: string | null;
    onInstanceChange: (instanceID: string) => void;
    onProjectChange: (projectID: string) => void;
    onError: (err: string) => void;

    theme: {};
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;

    installedInstances: Instance[];
    connectedInstances: Instance[];
    defaultUserInstance?: Instance;
    fetchJiraProjectMetadata: (instanceID: string) => Promise<{data?: ProjectMetadata; error: Error}>;
    hideProjectSelector?: boolean;
};

type State = {
    fetchingProjectMetadata: boolean;
    jiraProjectMetadata: ProjectMetadata | null;
    disableInstanceSelector: boolean;
};

export default class JiraInstanceAndProjectSelector extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);

        let instanceID = '';
        if (this.props.selectedInstanceID) {
            instanceID = this.props.selectedInstanceID;
        } else if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstance) {
            instanceID = this.props.defaultUserInstance.instance_id;
        }

        let fetchingProjectMetadata = false;
        if (instanceID) {
            this.props.onInstanceChange(instanceID);
            if (!props.hideProjectSelector) {
                this.fetchJiraProjectMetadata(instanceID);
                fetchingProjectMetadata = true;
            }
        }

        this.state = {
            fetchingProjectMetadata,
            jiraProjectMetadata: null,
            disableInstanceSelector: Boolean(this.props.selectedInstanceID),
        };
    }

    fetchJiraProjectMetadata = async (instanceID: string) => {
        this.setState({jiraProjectMetadata: null, fetchingProjectMetadata: true});
        const {data, error} = await this.props.fetchJiraProjectMetadata(instanceID);
        if (error) {
            this.setState({fetchingProjectMetadata: false});
            this.props.onError(error.message);
        } else {
            this.setState({jiraProjectMetadata: data as ProjectMetadata, fetchingProjectMetadata: false});
        }
    }

    handleJiraInstanceChange = (_: string, instanceID: string) => {
        if (instanceID === this.props.selectedInstanceID) {
            return;
        }

        this.props.onInstanceChange(instanceID);
        if (!this.props.hideProjectSelector) {
            this.fetchJiraProjectMetadata(instanceID);
        }
    }

    handleProjectChange = (_: string, projectID: string) => {
        this.props.onProjectChange(projectID);
    }

    render() {
        const instanceOptions: ReactSelectOption[] = this.props.installedInstances.map((instance: Instance) => (
            {label: instance.instance_id, value: instance.instance_id}
        ));

        const label = this.state.disableInstanceSelector ? 'Instance (saved)' : 'Instance';
        let instanceSelector;
        if (this.props.connectedInstances.length > 1 && this.props.installedInstances.length > 1) {
            instanceSelector = (
                <ReactSelectSetting
                    name={'instance'}
                    label={label}
                    options={instanceOptions}
                    onChange={this.handleJiraInstanceChange}
                    value={instanceOptions.find((opt) => opt.value === this.props.selectedInstanceID)}
                    isDisabled={this.state.disableInstanceSelector}
                    required={!this.state.disableInstanceSelector}
                    theme={this.props.theme}
                />
            );
        }

        let projectSelector;
        if (this.props.selectedInstanceID && !this.props.hideProjectSelector) {
            const projectOptions = getProjectValues(this.state.jiraProjectMetadata);
            projectSelector = (
                <ReactSelectSetting
                    name={'projects'}
                    label={'Project'}
                    limitOptions={true}
                    required={true}
                    onChange={this.handleProjectChange}
                    options={projectOptions}
                    isMulti={false}
                    theme={this.props.theme}
                    value={projectOptions.find((option) => option.value === this.props.selectedProjectID) || null}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    isLoading={this.state.fetchingProjectMetadata}
                />
            );
        }

        return (
            <React.Fragment>
                {instanceSelector}
                {projectSelector}
            </React.Fragment>
        );
    }
}

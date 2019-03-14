// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import DropDown from 'components/settings/dropdown';
import FormButton from 'components/form_button';
import Input from 'components/settings/input';
import Loading from 'components/loading';
import MultiSelect from 'components/settings/multiselect';

const initialState = {
    submitting: false,
    metadata: null,
    fields: {
        description: '',
        project: {
            key: '',
        },
        issuetype: {
            name: '',
        },
    },
    error: null,
};

export default class CreateIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        getMetadata: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = initialState;
    }

    componentWillReceiveProps(nextProps) {
        if (this.props.post !== nextProps.post && nextProps.post) {
            this.getMetadata(nextProps.post.message);
        }
    }

    getMetadata = (description) => {
        const {getMetadata} = this.props;
        const {fields} = this.state;
        getMetadata().then((meta) => {
            if (meta.error) {
                this.setState({error: meta.error.message});
                return;
            }

            console.log('create meta', meta.data);
            const nFields = {
                ...fields,
            };

            nFields.description = description;
            if (meta.data && meta.data.projects && meta.data.projects.length) {
                const pr = meta.data.projects[0];
                nFields.project.key = pr.key;
                nFields.issuetype.name = pr.issuetypes[0].name;
            }
            this.setState({
                metadata: meta.data,
                fields: nFields,
            });
        });
    };

    getProjectMeta = (projectKey) => {
        const {metadata} = this.state;
        if (metadata && metadata.projects) {
            return metadata.projects.find((p) => p.key === projectKey) || [];
        }

        return [];
    };

    getProjectIssueTypes = (projectKey) => {
        const project = this.getProjectMeta(projectKey);
        if (project.issuetypes) {
            return project.issuetypes.filter((i) => !i.subtask);
        }
        return [];
    };

    getFields = (projectKey, issueType) => {
        if (projectKey && issueType) {
            const issues = this.getProjectIssueTypes(projectKey);
            const issue = issues.find((i) => i.name === issueType);
            if (issue) {
                return Object.values(issue.fields).filter((f) => {
                    return (f.required || f.key === 'description') &&
                        f.key !== 'project' && f.key !== 'issuetype' && f.key !== 'reporter';
                });
            }
        }

        return [];
    };

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const {create, post} = this.props;
        const {fields} = this.state;

        const issue = {
            post_id: post.id,
            fields,
        };

        this.setState({submitting: true});

        create(issue).then((created) => {
            if (created.error) {
                this.setState({error: created.error.message, submitting: false});
                return;
            }
            this.handleClose(e);
        });
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleDescriptionChange = (e) => {
        const description = e.target.value;
        const {fields} = this.state;
        const nFields = {
            ...fields,
            description,
        };

        this.setState({fields: nFields});
    };

    handleSettingChange = (id, value) => {
        const {description, project} = this.state.fields;
        switch (id) {
        case 'selectProject': {
            const fields = {
                description,
                project: {
                    key: value,
                },
            };
            const issueTypes = this.getProjectIssueTypes(value);
            fields.issuetype = {
                name: issueTypes.length && issueTypes[0].name,
            };
            this.setState({fields});
            break;
        }
        case 'selectType': {
            const fields = {
                description,
                project,
                issuetype: {
                    name: value,
                },
            };
            this.setState({fields});
            break;
        }
        default: {
            const nFields = {...this.state.fields};
            nFields[id] = value;
            this.setState({fields: nFields});
            break;
        }
        }
    };

    renderFields = () => {
        const {fields} = this.state;
        const fieldsToRender = this.getFields(fields.project.key, fields.issuetype.name);

        return fieldsToRender.map((f) => {
            if (f.key === 'description') {
                return (
                    <Input
                        key={`${fields.issuetype.name}-${f.key}`}
                        id={f.key}
                        label={f.name}
                        type='textarea'
                        value={fields[f.key]}
                        onChange={this.handleSettingChange}
                        required={f.required}
                    />
                );
            }

            if (f.schema.type === 'string') {
                return (
                    <Input
                        key={`${fields.issuetype.name}-${f.key}`}
                        id={f.key}
                        label={f.name}
                        value={fields[f.key]}
                        onChange={this.handleSettingChange}
                        required={f.required}
                    />
                );
            }

            let value;
            if (f.hasDefaultValue && !fields[f.key]) {
                value = f.defaultValue && f.defaultValue.name;
            } else {
                value = fields[f.key];
            }

            if (f.allowedValues && f.allowedValues.length) {
                const options = f.allowedValues.map((o) => ({value: o.name, text: o.name}));

                return (
                    <MultiSelect
                        key={`${fields.issuetype.name}-${f.key}`}
                        id={f.key}
                        label={f.name}
                        options={options}
                        selected={value && value.map((v) => v.name)}
                        required={f.required}
                        onChange={this.handleSettingChange}
                    />
                );
            }

            return null;
        });
    };

    render() {
        const {post, visible, theme} = this.props;
        const {fields, error, metadata, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
        }

        let component;
        if (error) {
            console.error('render error', error);
        }

        if (!post || !metadata || !fields.project.key) {
            component = <Loading/>;
        } else {
            const projectsOption = (
                <DropDown
                    id='selectProject'
                    values={metadata.projects.map((p) => ({value: p.key, text: p.name}))}
                    value={fields.project.key}
                    label='Project'
                    required={true}
                    onChange={this.handleSettingChange}
                />
            );

            const issueTypes = (
                <DropDown
                    id='selectType'
                    values={this.getProjectIssueTypes(fields.project.key).map((i) => ({value: i.name, text: i.name}))}
                    value={fields.issuetype.name}
                    label='Issue Type'
                    required={true}
                    onChange={this.handleSettingChange}
                />
            );

            component = (
                <div style={style.modal}>
                    {projectsOption}
                    {issueTypes}
                    {this.renderFields()}
                    <br/>
                </div>
            );
        }

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={visible}
                onHide={this.handleClose}
                onExited={this.handleClose}
                bsSize='large'
            >
                <Modal.Header closeButton={true}>
                    <Modal.Title>
                        {'Create Jira Ticket'}
                    </Modal.Title>
                </Modal.Header>
                <form role='form'>
                    <Modal.Body ref='modalBody'>
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        <FormButton
                            btnClass='btn-default'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            btnClass='btn btn-primary'
                            saving={submitting}
                            onClick={this.handleCreate}
                        >
                            {'Create'}
                        </FormButton>
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme) => ({
    modal: {
        padding: '1em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});

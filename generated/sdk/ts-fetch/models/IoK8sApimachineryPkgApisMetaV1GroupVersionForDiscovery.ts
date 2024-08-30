/* tslint:disable */
/* eslint-disable */
/**
 * Kubernetes
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: v1.30.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { mapValues } from '../runtime';
/**
 * GroupVersion contains the "group/version" and "version" string of a version. It is made a struct to keep extensibility.
 * @export
 * @interface IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery
 */
export interface IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery {
    /**
     * groupVersion specifies the API group and version in the form "group/version"
     * @type {string}
     * @memberof IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery
     */
    groupVersion: string;
    /**
     * version specifies the version in the form of "version". This is to save the clients the trouble of splitting the GroupVersion.
     * @type {string}
     * @memberof IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery
     */
    version: string;
}

/**
 * Check if a given object implements the IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery interface.
 */
export function instanceOfIoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery(value: object): value is IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery {
    if (!('groupVersion' in value) || value['groupVersion'] === undefined) return false;
    if (!('version' in value) || value['version'] === undefined) return false;
    return true;
}

export function IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscoveryFromJSON(json: any): IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery {
    return IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscoveryFromJSONTyped(json, false);
}

export function IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscoveryFromJSONTyped(json: any, ignoreDiscriminator: boolean): IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery {
    if (json == null) {
        return json;
    }
    return {
        
        'groupVersion': json['groupVersion'],
        'version': json['version'],
    };
}

export function IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscoveryToJSON(value?: IoK8sApimachineryPkgApisMetaV1GroupVersionForDiscovery | null): any {
    if (value == null) {
        return value;
    }
    return {
        
        'groupVersion': value['groupVersion'],
        'version': value['version'],
    };
}


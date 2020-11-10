.. KubeEdge documentation master file, created by
   sphinx-quickstart on Fri Feb  8 12:12:47 2019.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to KubeEdge's documentation!
====================================


.. figure:: images/KubeEdge_logo.png
      :width: 150px
      :align: right
      :target: https://kubeedge.io

KubeEdge is an open source system for extending native containerized
application orchestration capabilities to hosts at Edge.

.. toctree::
   KubeEdge Home <https://kubeedge.io>

.. toctree::
   :maxdepth: 1
   :caption: Getting Started

   Why KubeEdge <components/kubeedge.md>
   getting-started
   roadmap

.. toctree::
   :maxdepth: 1
   :caption: Setup

   setup/keadm
   setup/local
   setup/upgrade

.. toctree::
   :maxdepth: 1
   :caption: Configuration

   KubeEdge <configuration/kubeedge>
   CRI <configuration/cri>
   Storage <configuration/storage>

.. toctree::
   :maxdepth: 1
   :caption: General Components

   components/beehive

.. toctree::
   :maxdepth: 1
   :caption: Cloud Components

   components/cloud/controller
   components/cloud/cloudhub
   components/cloud/device_controller

.. toctree::
   :maxdepth: 1
   :caption: Edge Components

   components/edge/edged
   components/edge/eventbus
   components/edge/metamanager
   components/edge/edgehub
   components/edge/devicetwin


.. toctree::
   :maxdepth: 1
   :caption: Edgesite

   EdgeSite <components/edgesite>

.. toctree::
   :maxdepth: 1
   :caption: Mappers

   Bluetooth <components/mappers/bluetooth_mapper>
   ModBus <components/mappers/modbus_mapper>

.. toctree::
   :maxdepth: 1
   :caption: Contributing

   contributing/contribute
   governance
   Maintainer <contributing/community>
   propoals
   contributing/feature-lifecycle

.. toctree::
   :maxdepth: 1
   :caption: Developer Guide

   Device Management <contributing/device_crd_guide>
   contributing/message_topics
   Unit Test <contributing/unit_test_guide>
   Bluetooth Mapper E2E Test <contributing/bluetooth_mapper_e2e_guide>
   contributing/edgemesh_guide
   Memory Footprint Test <contributing/memfootprint-test-setup>

.. toctree::
   :maxdepth: 1

   FAQ <troubleshooting>
